package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	wlogger "github.com/webitel/logger/pkg/client/v2"
	"github.com/webitel/storage/model"
	"github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
	"github.com/webitel/webitel-go-kit/pkg/watcher"
	"github.com/webitel/wlog"
	"strings"
	"time"
)

type WatcherObserver interface {
	GetId() string
	Update(watcher.EventType, map[string]any) error
}

type TriggerObserver[T any, V any] struct {
	amqpPublisher rabbitmq.Publisher
	id            string
	config        *model.TriggerWatcherSettings
	logger        *wlog.Logger
	converter     func(T) (V, error)
}

func NewTriggerObserver[T any, V any](
	amqpBroker rabbitmq.Broker,
	amqpPublisher rabbitmq.Publisher,
	config *model.TriggerWatcherSettings,
	conv func(T) (V, error),
	log *wlog.Logger,
) (*TriggerObserver[T, V], error) {
	opts := []rabbitmq.ExchangeOption{rabbitmq.WithDurable(true)}
	cfg, err := rabbitmq.NewExchangeConfig(config.ExchangeName, rabbitmq.ExchangeTypeTopic, opts...)
	if err != nil {
		return nil, err
	}

	if err := amqpBroker.DeclareExchange(context.Background(), cfg); err != nil {
		return nil, fmt.Errorf("could not create topic exchange %s: %w", config.ExchangeName, err)
	}

	return &TriggerObserver[T, V]{
		amqpPublisher: amqpPublisher,
		config:        config,
		id:            "Trigger Watcher",
		logger:        log,
		converter:     conv,
	}, nil
}

func (cao *TriggerObserver[T, V]) GetId() string {
	return cao.id
}

func (cao *TriggerObserver[T, V]) Update(et watcher.EventType, args map[string]any) error {
	obj, ok := args["obj"].(T)
	if !ok {
		return fmt.Errorf("expected obj of type %T, got %T", *new(T), args["obj"])
	}

	domainId, err := getDomainID(args)
	if err != nil {
		return fmt.Errorf("trigger update: %w", err)
	}

	message, err := cao.converter(obj)
	if err != nil {
		return fmt.Errorf("convert obj: %w", err)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	objStr, err := classifyObject(obj)
	if err != nil {
		return fmt.Errorf("classify object: %w", err)
	}

	routingKey := cao.getRoutingKeyByEventType("cases", objStr, et, domainId)
	cao.logger.Debug(fmt.Sprintf("Trying to publish message to %s", routingKey))

	return cao.amqpPublisher.Publish(context.Background(), routingKey, data, amqp091.Table{})
}

func (cao *TriggerObserver[T, V]) getRoutingKeyByEventType(
	service string,
	object string,
	eventType watcher.EventType,
	domainId int64,
) string {
	return fmt.Sprintf(
		"%s.%s.%s.%d",
		service,
		object,
		strings.Replace(cao.config.TopicName, "*", string(eventType), 1),
		domainId,
	)
}

type LoggerObserver struct {
	id      string
	logger  *wlogger.ObjectedLogger
	timeout time.Duration
}

func NewLoggerObserver(logger *wlogger.LoggerClient, objclass string, timeout time.Duration) (*LoggerObserver, error) {
	return &LoggerObserver{
		id:      fmt.Sprintf("%s logger", objclass),
		logger:  logger.GetObjectedLogger(objclass),
		timeout: timeout,
	}, nil
}

func (l *LoggerObserver) GetId() string {
	return l.id
}

func (l *LoggerObserver) Update(et watcher.EventType, args map[string]any) error {
	obj, ok := args["obj"]
	if !ok {
		return fmt.Errorf("'obj' not found in args")
	}

	file, ok := obj.(*model.File)
	if !ok {
		return fmt.Errorf("expected obj to be *model.File")
	}

	if file.UploadedBy == nil || file.UploadedBy.Id == 0 {
		return fmt.Errorf("missing uploader info in file.UploadedBy")
	}

	userId := file.UploadedBy.Id
	domainId := file.DomainId
	id := file.Id

	actionType := map[watcher.EventType]wlogger.Action{
		watcher.EventTypeCreate: wlogger.CreateAction,
		watcher.EventTypeDelete: wlogger.DeleteAction,
		watcher.EventTypeUpdate: wlogger.UpdateAction,
	}[et]

	if actionType == "" {
		return watcher.ErrUnknownType
	}

	// IP is unknown — pass empty string
	message, err := wlogger.NewMessage(int64(userId), "", actionType, id, file)
	if err != nil {
		return fmt.Errorf("create log message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	defer cancel()

	return l.logger.SendContext(ctx, domainId, message)
}

// Helpers

func getDomainID(args map[string]any) (int64, error) {
	if obj, ok := args["obj"]; ok {
		if file, ok := obj.(*model.File); ok {
			return file.DomainId, nil
		}
	}
	return 0, fmt.Errorf("domain_id not found in obj")
}

func classifyObject(obj any) (string, error) {
	switch obj.(type) {
	case *model.File:
		return model.PermissionScopeFiles, nil
	default:
		return "", fmt.Errorf("unsupported object type %T", obj)
	}
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"github.com/webitel/storage/model"
	wlogger "github.com/webitel/webitel-go-kit/infra/logger_client"
	"github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
	"github.com/webitel/webitel-go-kit/pkg/watcher"
	"github.com/webitel/wlog"
	"strconv"
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
	publisher rabbitmq.Publisher,
	config *model.TriggerWatcherSettings,
	conv func(T) (V, error),
	log *wlog.Logger,
) (*TriggerObserver[T, V], error) {
	return &TriggerObserver[T, V]{
		amqpPublisher: publisher,
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

	objStr, err := classifyTriggerObject(obj)
	if err != nil {
		return fmt.Errorf("classify object: %w", err)
	}

	routingKey := cao.getRoutingKeyByEventType("cases", objStr, et, domainId)
	cao.logger.Debug(fmt.Sprintf("trying to publish message to %s", routingKey))

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
		eventType,
		domainId,
	)
}

type LoggerObserver struct {
	id      string
	logger  *wlogger.ObjectedLogger
	timeout time.Duration
}

func NewLoggerObserver(logger *wlogger.Logger, objclass string, timeout time.Duration) (*LoggerObserver, error) {
	objLogger, err := logger.GetObjectedLogger(objclass)
	if err != nil {
		return nil, fmt.Errorf("failed to get objected logger for %s: %w", objclass, err)
	}

	return &LoggerObserver{
		id:      fmt.Sprintf("%s logger", objclass),
		logger:  objLogger,
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

	actionType := map[watcher.EventType]wlogger.Action{
		watcher.EventTypeCreate: wlogger.CreateAction,
		watcher.EventTypeDelete: wlogger.DeleteAction,
		watcher.EventTypeUpdate: wlogger.UpdateAction,
	}[et]

	if actionType == "" {
		return watcher.ErrUnknownType
	}

	message, err := wlogger.NewMessage(
		int64(file.UploadedBy.Id),
		"unknown",
		actionType,
		strconv.Itoa(int(file.Id)),
		file,
	)
	if err != nil {
		return fmt.Errorf("create log message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	defer cancel()

	_, err = l.logger.SendContext(ctx, file.DomainId, message)

	if err != nil {
		return fmt.Errorf("send log message: %w", err)
	}

	return nil
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

func classifyTriggerObject(obj any) (string, error) {
	switch v := obj.(type) {
	case *model.File:
		if v.Channel != nil {
			return fmt.Sprintf("%s_files", *v.Channel), nil
		}
		return model.PermissionScopeFiles, nil
	default:
		return "", fmt.Errorf("unsupported object type %T", obj)
	}
}

package rabbit

import (
	"context"
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/storage/broker/handler"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/store"
	"github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
)

const (
	CLUSTER_EVENT_DOMAIN_CREATE = "domains.create"
	EXCHANGE_WEBITEL            = "webitel"
)


type MultiEventConsumer struct {
	handlers map[string]func(ctx context.Context, msg amqp.Delivery) error
	logger   rabbitmq.Logger
}

func NewMultiEventConsumer(logger rabbitmq.Logger) *MultiEventConsumer {
	return &MultiEventConsumer{
		handlers: make(map[string]func(ctx context.Context, msg amqp.Delivery) error),
		logger:   logger,
	}
}

func (m *MultiEventConsumer) RegisterHandler(eventType string, handler func(ctx context.Context, msg amqp.Delivery) error) {
	m.handlers[eventType] = handler
}

func (m *MultiEventConsumer) Handle(ctx context.Context, msg amqp.Delivery) error {
    fullEventKey, ok := msg.Headers["event"].(string)
    if !ok || fullEventKey == "" {
        return model.NewBadRequestError(
            "consumer.multi_event_consumer.handle.header_check.missing_event_header", 
            "missing or invalid 'event' header in AMQP message.",
        )
    }
    
    parts := strings.Split(fullEventKey, ".")
    
    var baseEventType string
    if len(parts) >= 2 {
        baseEventType = strings.Join(parts[:2], ".") 
    } else {
        baseEventType = fullEventKey 
    }
    
    handler, exists := m.handlers[baseEventType]
    if !exists {
        m.logger.Info(fmt.Sprintf("no handler for base event type: %s (full key: %s)", baseEventType, fullEventKey))
        return nil
    }

    return handler(ctx, msg)
}

func SetupMultiEventConsumer(
	conn *rabbitmq.Connection,
	logger rabbitmq.Logger,
	store store.Store,
) (rabbitmq.Consumer, error) {
	ctx := context.Background()

	ch, err := conn.Channel(ctx)
	if err != nil {
		return nil, model.NewInternalError(
            "consumer.setup_multi_event_consumer.channel.failed_to_get_channel", 
            fmt.Sprintf("failed to get AMQP channel: %s", err.Error()),
        )
	}

	err = ch.ExchangeDeclare(EXCHANGE_WEBITEL, "topic", true, false, false, false, nil)
	if err != nil {
		return nil, model.NewInternalError(
            "consumer.setup_multi_event_consumer.exchange.failed_to_declare_exchange", 
            fmt.Sprintf("Failed to declare AMQP exchange '%s': %s", EXCHANGE_WEBITEL, err.Error()),
        )
	}

	queueName := "storage.domains.events.queue"
	queueCfg, err := rabbitmq.NewQueueConfig(queueName, rabbitmq.WithQueueDurable(true))
	if err != nil {
		return nil, model.NewInternalError(
            "consumer.setup_multi_event_consumer.queue_config.failed_to_create_config", 
            fmt.Sprintf("failed to create queue config for '%s': %s", queueName, err.Error()),
        )
	}

	_, err = ch.QueueDeclare(queueCfg.Name, queueCfg.Durable, false, false, false, nil)
	if err != nil {
		return nil, model.NewInternalError(
            "consumer.setup_multi_event_consumer.queue_declare.failed_to_declare_queue", 
            fmt.Sprintf("failed to declare AMQP queue '%s': %s", queueCfg.Name, err.Error()),
        )
	}

	err = ch.QueueBind(queueCfg.Name, "domains.*.*", EXCHANGE_WEBITEL, false, nil)
	if err != nil {
		return nil, model.NewInternalError(
            "consumer.setup_multi_event_consumer.queue_bind.failed_to_bind_queue", 
            fmt.Sprintf("failed to bind queue '%s' to exchange '%s': %s", queueCfg.Name, EXCHANGE_WEBITEL, err.Error()),
        )
	}

	consumerCfg, err := rabbitmq.NewConsumerConfig("multi-events-consumer",
		rabbitmq.WithConsumerMaxWorkers(10))
	if err != nil {
		return nil, model.NewInternalError(
            "consumer.setup_multi_event_consumer.consumer_config.failed_to_create_config", 
            fmt.Sprintf("failed to create consumer config: %s", err.Error()),
        )
	}

	multiConsumer := NewMultiEventConsumer(logger)
	
	multiConsumer.RegisterHandler(
		CLUSTER_EVENT_DOMAIN_CREATE,
		handler.NewAdapter(handler.NewEventDomainCreatedHandler(store.FilePolicies())),
	)
	
	consumer := rabbitmq.NewConsumer(conn, queueCfg, consumerCfg, multiConsumer.Handle, logger)
	return consumer, nil
}
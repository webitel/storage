package rabbit

import (
	"context"
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/storage/broker/handler"
	"github.com/webitel/storage/model"
	"github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
)

const (
	EXCHANGE_WEBITEL string = "webitel"
)


type MultiEventConsumer struct {
	handlers map[string]func(ctx context.Context, msg amqp.Delivery) error
	logger   rabbitmq.Logger
	rabbitConsumer rabbitmq.Consumer
	ch *amqp.Channel
}

func NewMultiEventConsumer(logger rabbitmq.Logger) *MultiEventConsumer {
	return &MultiEventConsumer{
		handlers: make(map[string]func(ctx context.Context, msg amqp.Delivery) error),
		logger:   logger,
	}
}

func (m *MultiEventConsumer) Start(ctx context.Context) error {
	return m.rabbitConsumer.Start(ctx)
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

func (m *MultiEventConsumer)SetupRabbitConsumer(conn *rabbitmq.Connection) (error) {
	var err error

	ctx := context.Background()
	m.ch, err = conn.Channel(ctx)
	if err != nil {
		return model.NewInternalError(
            "consumer.setup_multi_event_consumer.channel.failed_to_get_channel", 
            fmt.Sprintf("failed to get AMQP channel: %s", err.Error()),
        )
	}

	err = m.ch.ExchangeDeclare(EXCHANGE_WEBITEL, "topic", true, false, false, false, nil)
	if err != nil {
		return model.NewInternalError(
            "consumer.setup_multi_event_consumer.exchange.failed_to_declare_exchange", 
            fmt.Sprintf("Failed to declare AMQP exchange '%s': %s", EXCHANGE_WEBITEL, err.Error()),
        )
	}

	queueName := "storage.domains.events.queue"
	queueCfg, err := rabbitmq.NewQueueConfig(
		queueName, 
		rabbitmq.WithQueueDurable(true),
		rabbitmq.WithQueueArgument("x-queue-type", "quorum"),
	)

	if err != nil {
		return  model.NewInternalError(
            "consumer.setup_multi_event_consumer.queue_config.failed_to_create_config", 
            fmt.Sprintf("failed to create queue config for '%s': %s", queueName, err.Error()),
        )
	}

	_, err = m.ch.QueueDeclare(queueCfg.Name, queueCfg.Durable, false, false, false, queueCfg.Arguments)
	if err != nil {
		return  model.NewInternalError(
            "consumer.setup_multi_event_consumer.queue_declare.failed_to_declare_queue", 
            fmt.Sprintf("failed to declare AMQP queue '%s': %s", queueCfg.Name, err.Error()),
        )
	}

	err = m.ch.QueueBind(queueCfg.Name, "domains.*.*", EXCHANGE_WEBITEL, false, nil)
	if err != nil {
		return  model.NewInternalError(
            "consumer.setup_multi_event_consumer.queue_bind.failed_to_bind_queue", 
            fmt.Sprintf("failed to bind queue '%s' to exchange '%s': %s", queueCfg.Name, EXCHANGE_WEBITEL, err.Error()),
        )
	}

	consumerCfg, err := rabbitmq.NewConsumerConfig("multi-events-consumer",
		rabbitmq.WithConsumerMaxWorkers(10))
	if err != nil {
		return  model.NewInternalError(
            "consumer.setup_multi_event_consumer.consumer_config.failed_to_create_config", 
            fmt.Sprintf("failed to create consumer config: %s", err.Error()),
        )
	}
	
	m.rabbitConsumer = rabbitmq.NewConsumer(conn, queueCfg, consumerCfg, m.Handle, m.logger)

	return nil
}

func (m *MultiEventConsumer) Close(ctx context.Context) error {
    var errs []string

    if m.rabbitConsumer != nil {
        m.logger.Info("stopping rabbitmq consumer...")
        if err := m.rabbitConsumer.Shutdown(ctx); err != nil {
            errMsg := fmt.Sprintf("failed to close rabbitmq consumer: %s", err.Error())
            errs = append(errs, errMsg)
        }
    }

    if m.ch != nil {
        m.logger.Info("closing amqp channel...")
        if err := m.ch.Close(); err != nil {
            if !strings.Contains(err.Error(), "already closed") {
                errMsg := fmt.Sprintf("failed to close amqp channel: %s", err.Error())
                errs = append(errs, errMsg)
            }
        }
    }

    m.ch = nil
    m.rabbitConsumer = nil
    m.handlers = nil

    if len(errs) > 0 {
        return model.NewInternalError(
            "consumer.multi_event_consumer.close.failed_to_clean_resources",
            fmt.Sprintf("one or more errors occurred during resource cleanup: %s", strings.Join(errs, "; ")),
        )
    }

    m.logger.Info("multi-event consumer resources successfully closed.")
    return nil
}

func HandleFunc[T any](m *MultiEventConsumer, h handler.AppEventHandler[T]) {
	convertedHandler := handler.NewAdapter(h)
	m.RegisterHandler(h.Event(), convertedHandler)
}


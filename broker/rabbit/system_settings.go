package rabbit

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/storage/model"
	"github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
)

const (
	EXCHANGE_EVENT string = "event"

	systemSettingsRoutingKey    = model.SystemSettingsObjectName + ".#"
	systemSettingsReconnectWait = 5 * time.Second
)

var errSystemSettingsDeliveriesClosed = errors.New("system settings deliveries channel closed")

type SystemSettingsConsumer struct {
	conn    *rabbitmq.Connection
	handler func(e *model.SystemSettingEvent)
	logger  rabbitmq.Logger
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewSystemSettingsConsumer(conn *rabbitmq.Connection, handler func(e *model.SystemSettingEvent), logger rabbitmq.Logger) *SystemSettingsConsumer {
	return &SystemSettingsConsumer{
		conn:    conn,
		handler: handler,
		logger:  logger,
		done:    make(chan struct{}),
	}
}

func (c *SystemSettingsConsumer) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	go c.run(ctx)

	return nil
}

func (c *SystemSettingsConsumer) run(ctx context.Context) {
	defer close(c.done)

	for {
		if err := c.consume(ctx); err != nil {
			c.logger.Error("system settings consumer", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(systemSettingsReconnectWait):
		}
	}
}

func (c *SystemSettingsConsumer) consume(ctx context.Context) error {
	ch, err := c.conn.NewChannel(ctx)
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	if err = ch.ExchangeDeclare(EXCHANGE_EVENT, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %q: %w", EXCHANGE_EVENT, err)
	}

	queue := fmt.Sprintf("storage.system_settings.%s", model.NewId()[0:10])
	if _, err = ch.QueueDeclare(queue, false, true, true, false, nil); err != nil {
		return fmt.Errorf("declare queue %q: %w", queue, err)
	}

	if err = ch.QueueBind(queue, systemSettingsRoutingKey, EXCHANGE_EVENT, false, nil); err != nil {
		return fmt.Errorf("bind queue %q: %w", queue, err)
	}

	msgs, err := ch.Consume(queue, queue, false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume queue %q: %w", queue, err)
	}

	for {
		select {
		case <-ctx.Done():
			if err = ch.Cancel(queue, false); err != nil && !errors.Is(err, amqp.ErrClosed) {
				return fmt.Errorf("cancel consumer %q: %w", queue, err)
			}
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return errSystemSettingsDeliveriesClosed
			}

			if e, appErr := model.NewSystemSettingEventFromRoutingKey(msg.RoutingKey); appErr != nil {
				c.logger.Warn("invalid system settings event", "error", appErr.Error())
			} else {
				c.handler(e)
			}

			if err = msg.Ack(false); err != nil {
				c.logger.Error("ack system settings event", err)
			}
		}
	}
}

func (c *SystemSettingsConsumer) Close() {
	if c.cancel == nil {
		return
	}

	c.cancel()
	<-c.done
}

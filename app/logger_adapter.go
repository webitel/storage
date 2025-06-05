package app

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	broker "github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
)

// LoggerAdapter wraps a broker.Publisher and always publishes to the "logger" exchange.
type LoggerAdapter struct {
	publisher broker.Publisher
}

func (l *LoggerAdapter) Publish(ctx context.Context, routingKey string, body []byte, headers map[string]interface{}) error {
	amqpHeaders := amqp.Table{}
	for k, v := range headers {
		amqpHeaders[k] = v
	}
	return l.publisher.Publish(ctx, routingKey, body, amqpHeaders)
}

// NewLoggerAdapter constructs a LoggerAdapter with the given Publisher.
func NewLoggerAdapter(publisher broker.Publisher) *LoggerAdapter {
	return &LoggerAdapter{publisher: publisher}
}

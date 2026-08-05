package app

import (
	"context"

	broker "github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
)

// LoggerAdapter wraps a broker.Publisher and always publishes to the "logger" exchange.
type LoggerAdapter struct {
	publisher broker.Publisher
}

func (l *LoggerAdapter) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	return l.publisher.Publish(ctx, exchange, routingKey, body, nil)
}

// NewLoggerAdapter constructs a LoggerAdapter with the given Publisher.
func NewLoggerAdapter(publisher broker.Publisher) *LoggerAdapter {
	return &LoggerAdapter{publisher: publisher}
}

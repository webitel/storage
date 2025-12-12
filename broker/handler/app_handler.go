package handler

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AppEventHandler[T any] interface {
	Handle(ctx context.Context, event T) error
	Event() string
} 

func NewAdapter[T any](handler AppEventHandler[T]) func (ctx context.Context, msg amqp.Delivery) error {
	return  func (ctx context.Context, msg amqp.Delivery) error {
		var event T

		err := json.Unmarshal(msg.Body, &event)
		if err != nil {
			return err
		}

		return handler.Handle(ctx, event)
	}
}
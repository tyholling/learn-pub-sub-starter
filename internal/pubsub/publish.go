package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](channel *amqp.Channel, exchange, key string, val T) error {
	buf, err := json.Marshal(val)
	if err != nil {
		return err
	}
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        buf,
	}
	return channel.PublishWithContext(context.Background(), exchange, key, false, false, msg)
}

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

// SimpleQueueType represents the type of a queue: durable or transient.
type SimpleQueueType int

const (
	// QueueDurable indicates that the queue should survive broker restarts.
	QueueDurable SimpleQueueType = iota
	// QueueTransient indicates that the queue is temporary and will be deleted when no longer used.
	QueueTransient
)

func DeclareAndBind(
	conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType) (
	*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	isDurable := queueType == QueueDurable
	queue, err := channel.QueueDeclare(queueName, isDurable, !isDurable, !isDurable, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	return channel, queue, nil
}

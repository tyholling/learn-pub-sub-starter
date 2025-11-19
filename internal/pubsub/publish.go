// Package pubsub has publish/subscribe functions
package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

// PublishJSON sends a message to a channel
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

// QueueType represents the type of a queue: durable or transient.
type QueueType int

const (
	// QueueDurable indicates that the queue should survive broker restarts.
	QueueDurable QueueType = iota
	// QueueTransient indicates that the queue is temporary and will be deleted when no longer used.
	QueueTransient
)

// DeclareAndBind creates a queue and binds it to a channel
func DeclareAndBind(
	conn *amqp.Connection, exchange, queueName, key string, queueType QueueType,
) (*amqp.Channel, amqp.Queue, error) {

	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	isDurable := queueType == QueueDurable
	queue, err := channel.QueueDeclare(queueName, isDurable, !isDurable, !isDurable, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	if err := channel.QueueBind(queue.Name, key, exchange, false, amqp.Table{}); err != nil {
		return nil, amqp.Queue{}, err
	}
	return channel, queue, nil
}

// SubscribeJSON consumes messages from a queue
func SubscribeJSON[T any](
	conn *amqp.Connection, exchange, queueName, key string,
	queueType QueueType, handler func(T),
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	ch, err := channel.Consume(queue.Name, "", false, false, false, false, amqp.Table{})
	if err != nil {
		return err
	}
	go func() {
		for msg := range ch {
			var val T
			if err := json.Unmarshal(msg.Body, &val); err != nil {
				log.Errorf("failed to unmarshal %T to %T", msg.Body, val)
				log.Error(err)
			}
			log.Infof("received message: %s", string(msg.Body))
			handler(val)
			if err := msg.Ack(false); err != nil {
				log.Error(err)
			}
		}
	}()

	return nil
}

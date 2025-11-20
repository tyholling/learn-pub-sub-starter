// Package pubsub has publish/subscribe functions
package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

// QueueType indicates durable or transient
type QueueType int

const (
	// QueueDurable indicates durable
	QueueDurable QueueType = iota
	// QueueTransient indicates transient
	QueueTransient
)

// AckType indicates response behavior
type AckType int

const (
	// Ack indicates acknowledge
	Ack AckType = iota
	// NackRequeue indicates nack and requeue
	NackRequeue
	// NackDiscard indicates nack and discard
	NackDiscard
)

// DeclareAndBind creates a queue and binds it to a channel
func DeclareAndBind(
	connection *amqp.Connection, exchange, queueName, key string, queueType QueueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	channel.Qos(10, 0, false)

	isDurable := queueType == QueueDurable
	table := amqp.Table{"x-dead-letter-exchange": routing.ExchangePerilDeadLetter}
	queue, err := channel.QueueDeclare(queueName, isDurable, !isDurable, !isDurable, false, table)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	if err := channel.QueueBind(queue.Name, key, exchange, false, amqp.Table{}); err != nil {
		return nil, amqp.Queue{}, err
	}
	return channel, queue, nil
}

// PublishGob sends a gob message to a channel
func PublishGob[T any](channel *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(val); err != nil {
		log.Error(err)
		return err
	}

	msg := amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	}
	return channel.Publish(exchange, key, false, false, msg)
}

// PublishJSON sends a json message to a channel
func PublishJSON[T any](channel *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)

	if err := encoder.Encode(val); err != nil {
		log.Error(err)
		return err
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        buffer.Bytes(),
	}
	return channel.Publish(exchange, key, false, false, msg)
}

// SubscribeGob consumes gob messages from a queue
func SubscribeGob[T any](
	connection *amqp.Connection, exchange, queueName, key string,
	queueType QueueType, handler func(T) AckType,
) error {
	channel, queue, err := DeclareAndBind(connection, exchange, queueName, key, queueType)
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
			decoder := gob.NewDecoder(bytes.NewReader(msg.Body))
			if err := decoder.Decode(&val); err != nil {
				log.Error(err)
				continue
			}
			switch handler(val) {
			case Ack:
				if err := msg.Ack(false); err != nil {
					log.Error(err)
				} else {
					// log.Infof("ack: %s", msg.Body)
				}

			case NackRequeue:
				if err := msg.Nack(false, true); err != nil {
					log.Error(err)
				} else {
					// log.Infof("nack requeue: %s", msg.Body)
				}

			case NackDiscard:
				if err := msg.Nack(false, false); err != nil {
					log.Error(err)
				} else {
					// log.Infof("nack discard: %s", msg.Body)
				}
			}
		}
	}()

	return nil
}

// SubscribeJSON consumes messages from a queue
func SubscribeJSON[T any](
	connection *amqp.Connection, exchange, queueName, key string,
	queueType QueueType, handler func(T) AckType,
) error {
	channel, queue, err := DeclareAndBind(connection, exchange, queueName, key, queueType)
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
			decoder := json.NewDecoder(bytes.NewReader(msg.Body))
			if err := decoder.Decode(&val); err != nil {
				log.Error(err)
				continue
			}
			switch handler(val) {
			case Ack:
				if err := msg.Ack(false); err != nil {
					log.Error(err)
				} else {
					log.Infof("ack: %s", msg.Body)
				}

			case NackRequeue:
				if err := msg.Nack(false, true); err != nil {
					log.Error(err)
				} else {
					log.Infof("nack requeue: %s", msg.Body)
				}

			case NackDiscard:
				if err := msg.Nack(false, false); err != nil {
					log.Error(err)
				} else {
					log.Infof("nack discard: %s", msg.Body)
				}
			}
		}
	}()

	return nil
}

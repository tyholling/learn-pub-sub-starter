// Package pubsub has publish/subscribe functions
package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

type QueueType int

const (
	QueueDurable QueueType = iota
	QueueTransient
)

type AckType int

const (
	Ack = iota
	NackRequeue
	NackDiscard
)

// PublishJSON sends a json message to a channel
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

// PublishGob sends a gob message to a channel
func PublishGob[T any](channel *amqp.Channel, exchange, key string, val T) error {
	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	if err := encoder.Encode(val); err != nil {
		return err
	}
	msg := amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	}
	return channel.PublishWithContext(context.Background(), exchange, key, false, false, msg)
}

// DeclareAndBind creates a queue and binds it to a channel
func DeclareAndBind(
	conn *amqp.Connection, exchange, queueName, key string, queueType QueueType,
) (*amqp.Channel, amqp.Queue, error) {

	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

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

// SubscribeJSON consumes messages from a queue
func SubscribeJSON[T any](
	conn *amqp.Connection, exchange, queueName, key string,
	queueType QueueType, handler func(T) AckType,
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

// SubscribeGob consumes messages from a queue
func SubscribeGob[T any](
	conn *amqp.Connection, exchange, queueName, key string,
	queueType QueueType, handler func(T) AckType,
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
			buffer := bytes.NewBuffer(msg.Body)
			decoder := gob.NewDecoder(buffer)
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

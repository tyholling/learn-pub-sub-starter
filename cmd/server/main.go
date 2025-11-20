// server accepts connections from clients
package main

import (
	"fmt"
	"os"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	log.SetReportCaller(false)
}

func main() {
	log.Info("server started")
	defer log.Info("server stopped")

	connection, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Errorf("rabbitmq: failed to connect: %v", err)
		return
	}
	defer func() {
		if err := connection.Close(); err != nil {
			log.Errorf("rabbitmq: failed to close connection: %v", err)
		} else {
			log.Info("rabbitmq: connection closed")
		}
	}()
	log.Info("rabbitmq: connected")

	channel, err := connection.Channel()
	if err != nil {
		log.Errorf("rabbitmq: failed to open channel: %v", err)
		return
	}
	defer func() {
		if err := channel.Close(); err != nil {
			log.Errorf("rabbitmq: failed to close channel: %v", err)
		} else {
			log.Info("rabbitmq: channel closed")
		}
	}()
	log.Info("rabbitmq: channel open")

	if _, _, err := pubsub.DeclareAndBind(
		connection, routing.ExchangePerilTopic,
		routing.GameLogSlug, (routing.GameLogSlug + ".*"), pubsub.QueueDurable,
	); err != nil {
		log.Error(err)
	}

	// subscribe to game logs
	if err := pubsub.SubscribeGob(
		connection, routing.ExchangePerilTopic,
		routing.GameLogSlug, (routing.GameLogSlug + ".*"),
		pubsub.QueueDurable, handleLog(),
	); err != nil {
		log.Fatal(err)
	}

	gamelogic.PrintServerHelp()
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			log.Info("sending pause message")
			if err = pubsub.PublishJSON(
				channel, routing.ExchangePerilDirect, routing.PauseKey,
				routing.PlayingState{IsPaused: true},
			); err != nil {
				log.Errorf("failed to publish message: %v", err)
				return
			}

		case "resume":
			log.Info("sending resume message")
			if err = pubsub.PublishJSON(
				channel, routing.ExchangePerilDirect, routing.PauseKey,
				routing.PlayingState{IsPaused: false},
			); err != nil {
				log.Errorf("failed to publish message: %v", err)
				return
			}

		case "quit":
			return

		default:
			log.Infof("unexpected command: %s", words[0])
		}
	}
}

func handleLog() func(routing.GameLog) pubsub.AckType {
	return func(gl routing.GameLog) pubsub.AckType {
		defer fmt.Print("> ")
		if err := gamelogic.WriteLog(gl); err != nil {
			log.Errorf("failed to write log: %v", err)
		}
		return pubsub.Ack
	}
}

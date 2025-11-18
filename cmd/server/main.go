// server accepts connections from clients
package main

import (
	"encoding/json"
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

	gamelogic.PrintServerHelp()
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			log.Info("sending pause message")
			buf, err := json.Marshal(routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Errorf("failed to marshal json: %v", err)
				return
			}
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, buf)
			if err != nil {
				log.Errorf("failed to publish message: %v", err)
				return
			}
			log.Infof("published message: %s", string(buf))

		case "resume":
			log.Info("sending resume message")
			buf, err := json.Marshal(routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Errorf("failed to marshal json: %v", err)
				return
			}
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, buf)
			if err != nil {
				log.Errorf("failed to publish message: %v", err)
				return
			}
			log.Infof("published message: %s", string(buf))

		case "quit":
			return

		default:
			log.Infof("unexpected command: %s", words[0])
		}
	}
}

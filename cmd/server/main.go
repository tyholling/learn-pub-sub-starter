package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

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

	connectionString := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Errorf("failed to connect to rabbitmq: %v", err)
		return
	}
	defer func() {
		if err := connection.Close(); err != nil {
			log.Errorf("failed to close rabbitmq connection: %v", err)
		} else {
			log.Info("rabbitmq: connection closed")
		}
	}()
	log.Info("rabbitmq: connected")

	channel, err := connection.Channel()
	if err != nil {
		log.Errorf("failed to open channel: %v", err)
		return
	}

	playingState := routing.PlayingState{IsPaused: true}
	buf, err := json.Marshal(playingState)
	if err != nil {
		log.Errorf("failed to marshal json: %v")
		return
	}
	err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, buf)
	if err != nil {
		log.Errorf("failed to publish message: %v", err)
		return
	}
	log.Infof("published message: %s", string(buf))

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	defer fmt.Println()
}

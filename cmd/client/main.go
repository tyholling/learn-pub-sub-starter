// client connects to the server
package main

import (
	"fmt"
	"os"
	"os/signal"

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
	log.Info("client started")
	defer log.Info("client stopped")

	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Errorf("failed to connect to rabbitmq: %v", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("failed to close rabbitmq connection: %v", err)
		} else {
			log.Info("rabbitmq: connection closed")
		}
	}()
	log.Info("rabbitmq: connected")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Error(err)
		return
	}

	queueName := "pause." + username
	channel, queue, err := pubsub.DeclareAndBind(
		conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTransient)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("channel: %v", channel)
	log.Infof("queue: %v", queue)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	defer fmt.Println()
}

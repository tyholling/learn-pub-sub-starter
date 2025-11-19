// client connects to the server
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
	log.Info("client started")
	defer log.Info("client stopped")

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Errorf("rabbitmq: failed to connect: %v", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("rabbitmq: failed to close connection: %v", err)
		} else {
			log.Info("rabbitmq: connection closed")
		}
	}()
	log.Info("rabbitmq: connected")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Errorf("failed to get username: %v", err)
		return
	}

	queueName := "pause." + username
	_, _, err = pubsub.DeclareAndBind(
		conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTransient)
	if err != nil {
		log.Error(err)
		return
	}

	gameState := gamelogic.NewGameState(username)
	if err := pubsub.SubscribeJSON(
		conn, routing.ExchangePerilDirect, queueName,
		routing.PauseKey, pubsub.QueueTransient, handlerPause(gameState),
	); err != nil {
		log.Error(err)
		return
	}

	gamelogic.PrintClientHelp()
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			if err := gameState.CommandSpawn(words); err != nil {
				log.Errorf("failed to spawn: %v", err)
			}

		case "move":
			if _, err := gameState.CommandMove(words); err != nil {
				log.Errorf("failed to move: %v", err)
			}

		case "status":
			gameState.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming not allowed yet!")

		case "quit":
			gamelogic.PrintQuit()
			return

		default:
			log.Infof("unexpected command: %s", words[0])
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

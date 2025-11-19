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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Errorf("failed to get username: %v", err)
		return
	}
	gameState := gamelogic.NewGameState(username)

	// subscribe to pause events
	if err := pubsub.SubscribeJSON(
		connection, routing.ExchangePerilDirect,
		(routing.PauseKey + "." + username), routing.PauseKey,
		pubsub.QueueTransient, handlerPause(gameState),
	); err != nil {
		log.Error(err)
		return
	}

	// subscribe to army_moves.* events
	if err := pubsub.SubscribeJSON(
		connection, routing.ExchangePerilTopic,
		(routing.ArmyMovesPrefix + "." + username), (routing.ArmyMovesPrefix + ".*"),
		pubsub.QueueTransient, handlerMove(gameState, channel),
	); err != nil {
		log.Error(err)
		return
	}

	// subscribe to war events
	if err := pubsub.SubscribeJSON(
		connection, routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix, (routing.WarRecognitionsPrefix + "." + username),
		pubsub.QueueDurable, handlerWar(gameState),
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
			move, err := gameState.CommandMove(words)
			if err != nil {
				log.Errorf("failed to move: %v", err)
				break
			}
			log.Info("sending move message")
			routingKey := routing.ArmyMovesPrefix + "." + username
			if err = pubsub.PublishJSON(
				channel, routing.ExchangePerilTopic, routingKey, move,
			); err != nil {
				log.Errorf("failed to publish message: %v", err)
			} else {
				log.Info("published move")
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, channel *amqp.Channel,
) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)
		if outcome == gamelogic.MoveOutcomeMakeWar {
			if err := pubsub.PublishJSON(
				channel, routing.ExchangePerilTopic,
				(routing.WarRecognitionsPrefix + "." + move.Player.Username),
				gamelogic.RecognitionOfWar{Attacker: gs.Player, Defender: move.Player},
			); err != nil {
				log.Error(err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}
		if outcome == gamelogic.MoveOutComeSafe {
			return pubsub.Ack
		}
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon, gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			log.Errorf("unexpected outcome: %v", outcome)
			return pubsub.NackDiscard
		}
	}
}

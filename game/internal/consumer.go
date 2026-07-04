package internal

import (
	"encoding/json"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const gamePairsQueue = "game_pairs"

type gamePairMessage struct {
	GameID  uint64 `json:"game_id"`
	Player1 uint   `json:"player1_id"`
	Player2 uint   `json:"player2_id"`
}

func rabbitMQURL() string {
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		return v
	}
	return "amqp://admin:adminpassword@rabbitmq:5672/"
}

// StartPairConsumer consumes matched player pairs and creates games.
func StartPairConsumer() {
	go func() {
		for {
			runPairConsumer()
			time.Sleep(2 * time.Second)
		}
	}()
}

func runPairConsumer() {
	conn, err := amqp.Dial(rabbitMQURL())
	if err != nil {
		log.Printf("pair consumer: dial failed: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("pair consumer: channel failed: %v", err)
		return
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(gamePairsQueue, true, false, false, false, nil)
	if err != nil {
		log.Printf("pair consumer: declare queue failed: %v", err)
		return
	}

	msgs, err := ch.Consume(gamePairsQueue, "game-service", false, false, false, false, nil)
	if err != nil {
		log.Printf("pair consumer: consume failed: %v", err)
		return
	}

	for d := range msgs {
		var pair gamePairMessage
		if err := json.Unmarshal(d.Body, &pair); err != nil {
			log.Printf("pair consumer: invalid message: %v", err)
			_ = d.Nack(false, false)
			continue
		}
		if err := CreateGameFromPair(pair.GameID, pair.Player1, pair.Player2); err != nil {
			log.Printf("pair consumer: create game %d failed: %v", pair.GameID, err)
			_ = d.Nack(true, true)
			continue
		}
		log.Printf("pair consumer: created game %d for players %d and %d", pair.GameID, pair.Player1, pair.Player2)
		_ = d.Ack(false)
	}
}

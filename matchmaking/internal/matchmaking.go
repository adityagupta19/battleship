package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	matchmakingQueue = "matchmaking_queue"
	gamePairsQueue   = "game_pairs"
	redisPendingKey  = "matchmaking:pending"
	gameIDKey        = "game:id"
	requestExpiry    = 5 * time.Minute
	expireInterval   = 1 * time.Minute
	matchResultTTL   = 5 * time.Minute
)

var (
	rabbitMQURL = getEnv("RABBITMQ_URL", "amqp://admin:adminpassword@rabbitmq:5672/")
	redisAddr   = getEnv("REDIS_ADDR", "redis:6379")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type matchmakingMessage struct {
	UserID     uint   `json:"user_id"`
	EnqueuedAt string `json:"enqueued_at"`
}

type gamePairMessage struct {
	GameID   uint64 `json:"game_id"`
	Player1  uint   `json:"player1_id"`
	Player2  uint   `json:"player2_id"`
}

// MatchResult is returned to a player when matched.
type MatchResult struct {
	GameID     uint64
	OpponentID uint32
}

func getRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: redisAddr})
}

func matchWaitKey(userID uint) string {
	return fmt.Sprintf("match:wait:%d", userID)
}

// PushUserToQueue enqueues a user for matchmaking unless already pending.
func PushUserToQueue(userID uint) error {
	rdb := getRedis()
	defer rdb.Close()
	ctx := context.Background()

	cutoff := time.Now().Add(-requestExpiry).Unix()
	score, err := rdb.ZScore(ctx, redisPendingKey, fmt.Sprintf("%d", userID)).Result()
	if err == nil && score >= float64(cutoff) {
		log.Printf("User %d already in matchmaking queue", userID)
		return nil
	}

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(matchmakingQueue, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	now := time.Now()
	body, err := json.Marshal(matchmakingMessage{UserID: userID, EnqueuedAt: now.Format(time.RFC3339)})
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := ch.Publish("", matchmakingQueue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	if err := rdb.ZAdd(ctx, redisPendingKey, redis.Z{Score: float64(now.Unix()), Member: fmt.Sprintf("%d", userID)}).Err(); err != nil {
		log.Printf("warning: failed to add user %d to Redis pending set: %v", userID, err)
	}

	log.Printf("User %d pushed to queue %s", userID, matchmakingQueue)
	return nil
}

// WaitForMatch blocks until the user is paired or the context times out.
func WaitForMatch(ctx context.Context, userID uint) (*MatchResult, error) {
	rdb := getRedis()
	defer rdb.Close()

	key := matchWaitKey(userID)
	result, err := rdb.BLPop(ctx, requestExpiry, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("matchmaking timed out")
		}
		return nil, fmt.Errorf("waiting for match: %w", err)
	}
	if len(result) < 2 {
		return nil, fmt.Errorf("invalid match result")
	}

	var mr MatchResult
	if err := json.Unmarshal([]byte(result[1]), &mr); err != nil {
		return nil, fmt.Errorf("parse match result: %w", err)
	}
	return &mr, nil
}

func notifyMatch(rdb *redis.Client, ctx context.Context, userID uint, result MatchResult) {
	data, _ := json.Marshal(result)
	key := matchWaitKey(userID)
	_ = rdb.LPush(ctx, key, data).Err()
	_ = rdb.Expire(ctx, key, matchResultTTL).Err()
}

func allocateGameID(rdb *redis.Client, ctx context.Context) (uint64, error) {
	id, err := rdb.Incr(ctx, gameIDKey).Result()
	if err != nil {
		return 0, err
	}
	return uint64(id), nil
}

// StartWorkers launches background matchmaking workers.
func StartWorkers() {
	go createGamesWorker()
	go expireRequestsWorker()
}

func createGamesWorker() {
	for {
		runCreateGamesWorker()
		time.Sleep(2 * time.Second)
	}
}

func runCreateGamesWorker() {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Printf("createGamesWorker: RabbitMQ dial failed: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("createGamesWorker: channel failed: %v", err)
		return
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(matchmakingQueue, true, false, false, false, nil)
	if err != nil {
		log.Printf("createGamesWorker: declare matchmaking queue failed: %v", err)
		return
	}

	_, err = ch.QueueDeclare(gamePairsQueue, true, false, false, false, nil)
	if err != nil {
		log.Printf("createGamesWorker: declare game_pairs queue failed: %v", err)
		return
	}

	if err := ch.Qos(2, 0, false); err != nil {
		log.Printf("createGamesWorker: qos failed: %v", err)
		return
	}

	msgs, err := ch.Consume(matchmakingQueue, "create-games", false, false, false, false, nil)
	if err != nil {
		log.Printf("createGamesWorker: consume failed: %v", err)
		return
	}

	rdb := getRedis()
	defer rdb.Close()
	ctx := context.Background()
	cutoff := time.Now().Add(-requestExpiry)
	var pending *matchmakingMessage

	for d := range msgs {
		var msg matchmakingMessage
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("createGamesWorker: invalid message: %v", err)
			_ = d.Nack(false, false)
			continue
		}

		enqueued, err := time.Parse(time.RFC3339, msg.EnqueuedAt)
		if err != nil {
			_ = d.Nack(false, false)
			continue
		}
		if enqueued.Before(cutoff) {
			_ = d.Ack(false)
			_, _ = rdb.ZRem(ctx, redisPendingKey, fmt.Sprintf("%d", msg.UserID)).Result()
			continue
		}

		score, err := rdb.ZScore(ctx, redisPendingKey, fmt.Sprintf("%d", msg.UserID)).Result()
		if err == redis.Nil || score < float64(cutoff.Unix()) {
			_ = d.Ack(false)
			continue
		}

		if pending == nil {
			pending = &msg
			_ = d.Ack(false)
			continue
		}

		gameID, err := allocateGameID(rdb, ctx)
		if err != nil {
			log.Printf("createGamesWorker: allocate game id failed: %v", err)
			_ = d.Nack(true, true)
			pending = nil
			continue
		}

		pair := gamePairMessage{
			GameID:  gameID,
			Player1: pending.UserID,
			Player2: msg.UserID,
		}
		pairBody, _ := json.Marshal(pair)
		if err := ch.Publish("", gamePairsQueue, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        pairBody,
		}); err != nil {
			log.Printf("createGamesWorker: publish game pair failed: %v", err)
			_ = d.Nack(true, true)
			pending = nil
			continue
		}

		_, _ = rdb.ZRem(ctx, redisPendingKey,
			fmt.Sprintf("%d", pending.UserID),
			fmt.Sprintf("%d", msg.UserID),
		).Result()

		notifyMatch(rdb, ctx, pending.UserID, MatchResult{GameID: gameID, OpponentID: uint32(msg.UserID)})
		notifyMatch(rdb, ctx, msg.UserID, MatchResult{GameID: gameID, OpponentID: uint32(pending.UserID)})

		log.Printf("createGamesWorker: paired users %d and %d -> game %d", pending.UserID, msg.UserID, gameID)
		_ = d.Ack(false)
		pending = nil
	}
}

func expireRequestsWorker() {
	rdb := getRedis()
	defer rdb.Close()
	ctx := context.Background()
	ticker := time.NewTicker(expireInterval)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-requestExpiry).Unix()
		n, err := rdb.ZRemRangeByScore(ctx, redisPendingKey, "-inf", fmt.Sprintf("%d", cutoff)).Result()
		if err != nil {
			log.Printf("expireRequestsWorker: ZRemRangeByScore failed: %v", err)
			continue
		}
		if n > 0 {
			log.Printf("expireRequestsWorker: removed %d expired pending request(s)", n)
		}
	}
}

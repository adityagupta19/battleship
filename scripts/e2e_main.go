package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	userpb "github.com/adityagupta19/battleship/user/gen/proto/userpb"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var defaultShips = []map[string]interface{}{
	{"type": "carrier", "start_x": 0, "start_y": 0, "horizontal": true},
	{"type": "battleship", "start_x": 2, "start_y": 0, "horizontal": true},
	{"type": "cruiser", "start_x": 4, "start_y": 0, "horizontal": true},
	{"type": "submarine", "start_x": 6, "start_y": 0, "horizontal": true},
	{"type": "destroyer", "start_x": 8, "start_y": 0, "horizontal": true},
}

func main() {
	userAddr := env("USER_ADDR", "localhost:50051")
	gatewayWS := env("GATEWAY_WS", "ws://localhost:8080/ws")

	u1 := registerUser(userAddr, "alice")
	u2 := registerUser(userAddr, "bob")
	log.Printf("registered users: %d, %d", u1, u2)

	var wg sync.WaitGroup
	var gameID uint64
	wg.Add(2)

	go func() {
		defer wg.Done()
		gid := playClient(gatewayWS, u1, true)
		if gid > 0 {
			gameID = gid
		}
	}()
	go func() {
		defer wg.Done()
		gid := playClient(gatewayWS, u2, false)
		if gid > 0 {
			gameID = gid
		}
	}()

	wg.Wait()
	log.Printf("e2e test completed, game_id=%d", gameID)
}

func registerUser(addr, username string) uint32 {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("user dial: %v", err)
	}
	defer conn.Close()
	client := userpb.NewUserServiceClient(conn)
	resp, err := client.RegisterUser(context.Background(), &userpb.RegisterRequest{Username: username})
	if err != nil {
		log.Fatalf("register %s: %v", username, err)
	}
	return resp.GetUserId()
}

func playClient(gatewayWS string, userID uint32, fireFirst bool) uint64 {
	u, _ := url.Parse(gatewayWS)
	q := u.Query()
	q.Set("user_id", fmt.Sprintf("%d", userID))
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("ws dial user %d: %v", userID, err)
	}
	defer conn.Close()

	send(conn, map[string]interface{}{"type": "find_match"})
	msg := recv(conn)
	if msg["type"] != "match_found" {
		log.Fatalf("user %d expected match_found, got %v", userID, msg)
	}
	gameID := uint64(msg["game_id"].(float64))
	log.Printf("user %d matched game %d opponent %v", userID, gameID, msg["opponent_id"])

	send(conn, map[string]interface{}{
		"type": "place_ships", "game_id": gameID, "ships": defaultShips,
	})
	msg = recv(conn)
	log.Printf("user %d place_ships: %v", userID, msg["type"])

	// wait for both ready
	for i := 0; i < 5; i++ {
		send(conn, map[string]interface{}{"type": "get_state", "game_id": gameID})
		msg = recv(conn)
		if msg["status"] == "active" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if fireFirst {
		send(conn, map[string]interface{}{"type": "fire_shot", "game_id": gameID, "x": 0, "y": 0})
		msg = recv(conn)
		log.Printf("user %d fire_shot: %v", userID, msg)
	}

	return gameID
}

func send(conn *websocket.Conn, v interface{}) {
	data, _ := json.Marshal(v)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Fatalf("write: %v", err)
	}
}

func recv(conn *websocket.Conn) map[string]interface{} {
	_, data, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	var msg map[string]interface{}
	_ = json.Unmarshal(data, &msg)
	return msg
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

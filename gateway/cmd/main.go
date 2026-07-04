package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	gamepb "github.com/adityagupta19/battleship/gateway/gen/proto/gamepb"
	mmpb "github.com/adityagupta19/battleship/gateway/gen/proto/matchmakingpb"
	userpb "github.com/adityagupta19/battleship/gateway/gen/proto/userpb"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type connHub struct {
	mu    sync.RWMutex
	conns map[uint]*websocket.Conn
}

func newConnHub() *connHub {
	return &connHub{conns: make(map[uint]*websocket.Conn)}
}

func (h *connHub) set(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[userID] = conn
}

func (h *connHub) remove(userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, userID)
}

type gateway struct {
	hub  *connHub
	mm   mmpb.MatchmakingServiceClient
	game gamepb.GameServiceClient
	user userpb.UserServiceClient
}

type clientMessage struct {
	Type    string          `json:"type"`
	GameID  uint64          `json:"game_id,omitempty"`
	Ships   []shipPlacement `json:"ships,omitempty"`
	X       int32           `json:"x,omitempty"`
	Y       int32           `json:"y,omitempty"`
}

type shipPlacement struct {
	Type       string `json:"type"`
	StartX     int32  `json:"start_x"`
	StartY     int32  `json:"start_y"`
	Horizontal bool   `json:"horizontal"`
}

type serverMessage struct {
	Type        string      `json:"type"`
	GameID      uint64      `json:"game_id,omitempty"`
	OpponentID  uint32      `json:"opponent_id,omitempty"`
	Status      string      `json:"status,omitempty"`
	Result      string      `json:"result,omitempty"`
	SunkShip    string      `json:"sunk_ship,omitempty"`
	GameOver    bool        `json:"game_over,omitempty"`
	WinnerID    uint32      `json:"winner_id,omitempty"`
	NextTurn    uint32      `json:"next_turn,omitempty"`
	YourTurn    bool        `json:"your_turn,omitempty"`
	YouReady    bool        `json:"you_ready,omitempty"`
	OppReady    bool        `json:"opponent_ready,omitempty"`
	YourBoard   interface{} `json:"your_board,omitempty"`
	OppView     interface{} `json:"opponent_view,omitempty"`
	Message     string      `json:"message,omitempty"`
}

func (g *gateway) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	resp, err := g.user.RegisterUser(r.Context(), &userpb.RegisterRequest{Username: req.Username})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]uint32{"user_id": resp.GetUserId()})
}

func withCORS(origin string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (g *gateway) handleWS(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil || userID == 0 {
		http.Error(w, "user_id query param required", http.StatusBadRequest)
		return
	}
	uid := uint(userID)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	g.hub.set(uid, conn)
	defer g.hub.remove(uid)

	log.Printf("user %d connected", uid)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("user %d disconnected: %v", uid, err)
			return
		}

		var msg clientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			g.send(conn, serverMessage{Type: "error", Message: "invalid json"})
			continue
		}

		switch msg.Type {
		case "find_match":
			g.handleFindMatch(conn, uid)
		case "place_ships":
			g.handlePlaceShips(conn, uid, msg)
		case "fire_shot":
			g.handleFireShot(conn, uid, msg)
		case "get_state":
			g.handleGetState(conn, uid, msg)
		default:
			g.send(conn, serverMessage{Type: "error", Message: "unknown message type"})
		}
	}
}

func (g *gateway) handleFindMatch(conn *websocket.Conn, uid uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := g.mm.FindMatch(ctx, &mmpb.FindMatchRequest{UserId: uint32(uid)})
	if err != nil {
		g.send(conn, serverMessage{Type: "error", Message: err.Error()})
		return
	}
	g.send(conn, serverMessage{
		Type:       "match_found",
		GameID:     resp.GetGameId(),
		OpponentID: resp.GetOpponentId(),
	})
}

func (g *gateway) handlePlaceShips(conn *websocket.Conn, uid uint, msg clientMessage) {
	var ships []*gamepb.ShipPlacement
	for _, s := range msg.Ships {
		ships = append(ships, &gamepb.ShipPlacement{
			Type:       s.Type,
			StartX:     s.StartX,
			StartY:     s.StartY,
			Horizontal: s.Horizontal,
		})
	}
	resp, err := g.game.PlaceShips(context.Background(), &gamepb.PlaceShipsRequest{
		GameId: msg.GameID,
		UserId: uint32(uid),
		Ships:  ships,
	})
	if err != nil {
		g.send(conn, serverMessage{Type: "error", Message: err.Error()})
		return
	}
	if !resp.GetSuccess() {
		g.send(conn, serverMessage{Type: "error", Message: resp.GetMessage()})
		return
	}
	g.send(conn, serverMessage{Type: "game_state", Status: resp.GetStatus(), GameID: msg.GameID})
	g.handleGetState(conn, uid, msg)
}

func (g *gateway) handleFireShot(conn *websocket.Conn, uid uint, msg clientMessage) {
	resp, err := g.game.FireShot(context.Background(), &gamepb.FireShotRequest{
		GameId: msg.GameID,
		UserId: uint32(uid),
		X:      msg.X,
		Y:      msg.Y,
	})
	if err != nil {
		g.send(conn, serverMessage{Type: "error", Message: err.Error()})
		return
	}

	out := serverMessage{
		Type:     "shot_result",
		GameID:   msg.GameID,
		Result:   resp.GetResult(),
		SunkShip: resp.GetSunkShip(),
		GameOver: resp.GetGameOver(),
		WinnerID: resp.GetWinnerId(),
		NextTurn: resp.GetNextTurn(),
	}
	g.send(conn, out)

	if resp.GetGameOver() {
		g.send(conn, serverMessage{Type: "game_over", GameID: msg.GameID, WinnerID: resp.GetWinnerId()})
	} else {
		g.handleGetState(conn, uid, msg)
	}
}

func (g *gateway) handleGetState(conn *websocket.Conn, uid uint, msg clientMessage) {
	resp, err := g.game.GetGameState(context.Background(), &gamepb.GetGameRequest{
		GameId: msg.GameID,
		UserId: uint32(uid),
	})
	if err != nil {
		g.send(conn, serverMessage{Type: "error", Message: err.Error()})
		return
	}
	g.send(conn, serverMessage{
		Type:      "game_state",
		GameID:    resp.GetGameId(),
		Status:    resp.GetStatus(),
		YourTurn:  resp.GetYourTurn(),
		YouReady:  resp.GetYouReady(),
		OppReady:  resp.GetOpponentReady(),
		WinnerID:  resp.GetWinnerId(),
		NextTurn:  resp.GetCurrentTurn(),
		YourBoard: resp.GetYourBoard(),
		OppView:   resp.GetOpponentView(),
	})
}

func (g *gateway) send(conn *websocket.Conn, msg serverMessage) {
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("ws write failed: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	mmAddr := envOr("MATCHMAKING_ADDR", "localhost:50052")
	gameAddr := envOr("GAME_ADDR", "localhost:50053")
	userAddr := envOr("USER_ADDR", "localhost:50051")
	corsOrigin := envOr("CORS_ORIGIN", "http://localhost:3000")
	port := envOr("PORT", "8080")

	mmConn, err := grpc.NewClient(mmAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("matchmaking dial: %v", err)
	}
	defer mmConn.Close()

	gameConn, err := grpc.NewClient(gameAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("game dial: %v", err)
	}
	defer gameConn.Close()

	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("user dial: %v", err)
	}
	defer userConn.Close()

	gw := &gateway{
		hub:  newConnHub(),
		mm:   mmpb.NewMatchmakingServiceClient(mmConn),
		game: gamepb.NewGameServiceClient(gameConn),
		user: userpb.NewUserServiceClient(userConn),
	}

	http.HandleFunc("/ws", withCORS(corsOrigin, gw.handleWS))
	http.HandleFunc("/api/register", withCORS(corsOrigin, gw.handleRegister))
	log.Printf("gateway listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("gateway failed: %v", err)
	}
}

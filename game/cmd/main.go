package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	gamepb "github.com/adityagupta19/battleship/game/gen/proto/gamepb"
	"github.com/adityagupta19/battleship/game/db"
	"github.com/adityagupta19/battleship/game/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	gamepb.UnimplementedGameServiceServer
}

func (s *server) GetGameState(_ context.Context, req *gamepb.GetGameRequest) (*gamepb.GameStateResponse, error) {
	database := db.GetDB()
	var game internal.Game
	if err := database.First(&game, req.GetGameId()).Error; err != nil {
		return nil, status.Error(codes.NotFound, "game not found")
	}

	userID := uint(req.GetUserId())
	self, opponent, _, err := playerSlotFromGame(&game, userID)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	resp := &gamepb.GameStateResponse{
		GameId:         game.ID,
		Status:         game.Status,
		CurrentTurn:    uint32(game.CurrentTurn),
		YourTurn:       uint(game.CurrentTurn) == userID,
		YouReady:       self.Ready,
		OpponentReady:  opponent.Ready,
		YourBoard:      buildBoardProto(self.Ships, self.ShotsReceived, true),
		OpponentView:   buildBoardProto(opponent.Ships, opponent.ShotsReceived, false),
	}
	if game.WinnerID != nil {
		resp.WinnerId = uint32(*game.WinnerID)
	}
	return resp, nil
}

func (s *server) PlaceShips(_ context.Context, req *gamepb.PlaceShipsRequest) (*gamepb.PlaceShipsResponse, error) {
	var placements []internal.ShipPlacementInput
	for _, sp := range req.GetShips() {
		placements = append(placements, internal.ShipPlacementInput{
			Type:       sp.GetType(),
			StartX:     int(sp.GetStartX()),
			StartY:     int(sp.GetStartY()),
			Horizontal: sp.GetHorizontal(),
		})
	}

	game, err := internal.PlaceShips(req.GetGameId(), uint(req.GetUserId()), placements)
	if err != nil {
		return &gamepb.PlaceShipsResponse{Success: false, Message: err.Error()}, nil
	}
	return &gamepb.PlaceShipsResponse{Success: true, Status: game.Status}, nil
}

func (s *server) FireShot(_ context.Context, req *gamepb.FireShotRequest) (*gamepb.FireShotResponse, error) {
	game, fr, err := internal.FireShot(req.GetGameId(), uint(req.GetUserId()), int(req.GetX()), int(req.GetY()))
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	resp := &gamepb.FireShotResponse{
		Result:    fr.Result,
		SunkShip:  fr.SunkShip,
		GameOver:  fr.GameOver,
		WinnerId:  uint32(fr.WinnerID),
		NextTurn:  uint32(fr.NextTurn),
	}
	_ = game
	return resp, nil
}

func playerSlotFromGame(game *internal.Game, userID uint) (*internal.PlayerState, *internal.PlayerState, bool, error) {
	state := game.ParsedState()
	if uint(game.Player1ID) == userID {
		return &state.Player1, &state.Player2, true, nil
	}
	if uint(game.Player2ID) == userID {
		return &state.Player2, &state.Player1, false, nil
	}
	return nil, nil, false, internal.ErrNotInGame
}

func buildBoardProto(ships []internal.Ship, shots []internal.Coord, showShips bool) *gamepb.BoardView {
	bv := &gamepb.BoardView{Grid: internal.BuildBoardView(ships, shots, showShips)}
	for _, sv := range internal.BuildShipViews(ships, showShips) {
		ship := &gamepb.ShipView{Type: sv.Type, Hits: sv.Hits, Sunk: sv.Sunk}
		for _, p := range sv.Positions {
			ship.Positions = append(ship.Positions, &gamepb.Coordinate{X: p.X, Y: p.Y})
		}
		bv.Ships = append(bv.Ships, ship)
	}
	return bv
}

var port = flag.Int("port", 50053, "The server port")

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	if envPort, exists := os.LookupEnv("PORT"); exists {
		fmt.Sscanf(envPort, "%d", port)
	}

	db.Connect()
	internal.StartPairConsumer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	gamepb.RegisterGameServiceServer(s, &server{})
	log.Printf("game service listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	mmpb "github.com/adityagupta19/battleship/matchmaking/gen/proto/matchmakingpb"
	"github.com/adityagupta19/battleship/matchmaking/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	mmpb.UnimplementedMatchmakingServiceServer
}

func (s *server) FindMatch(ctx context.Context, req *mmpb.FindMatchRequest) (*mmpb.FindMatchResponse, error) {
	userID := uint(req.GetUserId())
	if userID == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := internal.PushUserToQueue(userID); err != nil {
		return nil, status.Errorf(codes.Internal, "enqueue failed: %v", err)
	}

	result, err := internal.WaitForMatch(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.DeadlineExceeded, "matchmaking failed: %v", err)
	}

	return &mmpb.FindMatchResponse{
		GameId:     result.GameID,
		OpponentId: result.OpponentID,
	}, nil
}

var port = flag.Int("port", 50052, "The server port")

func main() {
	flag.Parse()

	if envPort, exists := os.LookupEnv("PORT"); exists {
		fmt.Sscanf(envPort, "%d", port)
	}

	internal.StartWorkers()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	mmpb.RegisterMatchmakingServiceServer(s, &server{})
	log.Printf("matchmaking service listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

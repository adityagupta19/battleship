package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	userpb "github.com/adityagupta19/battleship/user/gen/proto/userpb"
	internal "github.com/adityagupta19/battleship/user/internal"
	"google.golang.org/grpc"
)

type server struct {
	userpb.UnimplementedUserServiceServer
}

func (*server) RegisterUser(_ context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	user, err := internal.RegisterUser(req.GetUsername())
	if err != nil {
		return &userpb.RegisterResponse{}, err
	}
	resp := userpb.RegisterResponse{UserId: uint32(user.ID)}

	return &resp, nil
}

func (*server) GetUser(_ context.Context, req *userpb.UserRequest) (*userpb.UserResponse, error) {
	user, err := internal.GetUser(uint(req.GetUserId()))
	if err != nil {
		return &userpb.UserResponse{}, err
	}
	return &userpb.UserResponse{
		UserId:    uint32(user.ID),
		Username:  user.Username,
		Rating:    int32(user.Rating),
		CreatedAt: user.CreatedAt.Format(time.RFC3339), // Convert time.Time to string
	}, nil
}

var port = flag.Int("port", 50051, "The server port")

func main() {
	flag.Parse()

	// Use environment variable if set (for Docker)
	if envPort, exists := os.LookupEnv("PORT"); exists {
		fmt.Sscanf(envPort, "%d", port)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

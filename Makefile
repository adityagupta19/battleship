# Define variables for proto paths and output directories
PROTO_DIR := proto
USER_SERVICE_GEN := user/gen
MATCHMAKING_SERVICE_GEN := matchmaking/gen
GAME_SERVICE_GEN := game/gen
GATEWAY_GEN := gateway/gen

# Proto files
PROTO_FILES := $(PROTO_DIR)/common/game_state.proto \
               $(PROTO_DIR)/user/user.proto \
               $(PROTO_DIR)/matchmaking/matchmaking.proto \
               $(PROTO_DIR)/game/game.proto

# Default target
all: generate

# Generate Go code from proto files
generate:
	mkdir -p $(USER_SERVICE_GEN)/proto/common \
	         $(USER_SERVICE_GEN)/proto/userpb \
	         $(USER_SERVICE_GEN)/proto/matchmakingpb \
	         $(USER_SERVICE_GEN)/proto/gamepb
	mkdir -p $(MATCHMAKING_SERVICE_GEN)/proto/common \
	         $(MATCHMAKING_SERVICE_GEN)/proto/userpb \
	         $(MATCHMAKING_SERVICE_GEN)/proto/matchmakingpb \
	         $(MATCHMAKING_SERVICE_GEN)/proto/gamepb
	mkdir -p $(GAME_SERVICE_GEN)/proto/common \
	         $(GAME_SERVICE_GEN)/proto/userpb \
	         $(GAME_SERVICE_GEN)/proto/matchmakingpb \
	         $(GAME_SERVICE_GEN)/proto/gamepb
	mkdir -p $(GATEWAY_GEN)/proto/common \
	         $(GATEWAY_GEN)/proto/userpb \
	         $(GATEWAY_GEN)/proto/matchmakingpb \
	         $(GATEWAY_GEN)/proto/gamepb

	protoc --proto_path=$(PROTO_DIR) \
	       --go_out=$(USER_SERVICE_GEN)/proto \
	       --go-grpc_out=$(USER_SERVICE_GEN)/proto \
	       --go_out=$(MATCHMAKING_SERVICE_GEN)/proto \
	       --go-grpc_out=$(MATCHMAKING_SERVICE_GEN)/proto \
	       --go_out=$(GAME_SERVICE_GEN)/proto \
	       --go-grpc_out=$(GAME_SERVICE_GEN)/proto \
	       --go_out=$(GATEWAY_GEN)/proto \
	       --go-grpc_out=$(GATEWAY_GEN)/proto \
	       $(PROTO_FILES)

clean:
	rm -rf $(USER_SERVICE_GEN)/* $(MATCHMAKING_SERVICE_GEN)/* $(GAME_SERVICE_GEN)/* $(GATEWAY_GEN)/*

.PHONY: all generate clean

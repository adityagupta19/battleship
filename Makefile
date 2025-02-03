# Define variables for proto paths and output directories
PROTO_DIR := proto
USER_SERVICE_GEN := user/gen
MATCHMAKING_SERVICE_GEN := matchmaking/gen
GAME_SERVICE_GEN := game/gen

# Proto files
PROTO_FILES := $(PROTO_DIR)/common/game_state.proto \
               $(PROTO_DIR)/user/user.proto \
               $(PROTO_DIR)/matchmaking/matchmaking.proto

# Go plugin for protoc
PROTOC_GEN_GO := protoc-gen-go
PROTOC_GEN_GRPC := protoc-gen-go-grpc

# Default target
all: generate

# Generate Go code from proto files
generate:
	# Create the gen directories if they don't exist
	mkdir -p $(USER_SERVICE_GEN)/proto/common \
	         $(USER_SERVICE_GEN)/proto/user \
	         $(USER_SERVICE_GEN)/proto/matchmaking
	mkdir -p $(MATCHMAKING_SERVICE_GEN)/proto/common \
	         $(MATCHMAKING_SERVICE_GEN)/proto/user \
	         $(MATCHMAKING_SERVICE_GEN)/proto/matchmaking
	mkdir -p $(GAME_SERVICE_GEN)/proto/common \
	         $(GAME_SERVICE_GEN)/proto/user \
	         $(GAME_SERVICE_GEN)/proto/matchmaking

	# Run protoc to generate Go files
	protoc --proto_path=$(PROTO_DIR) \
	       --go_out=$(USER_SERVICE_GEN)/proto \
	       --go-grpc_out=$(USER_SERVICE_GEN)/proto \
	       --go_out=$(MATCHMAKING_SERVICE_GEN)/proto \
	       --go-grpc_out=$(MATCHMAKING_SERVICE_GEN)/proto \
	       --go_out=$(GAME_SERVICE_GEN)/proto \
	       --go-grpc_out=$(GAME_SERVICE_GEN)/proto \
	       $(PROTO_FILES)

# Clean generated files
clean:
	rm -rf $(USER_SERVICE_GEN)/* $(MATCHMAKING_SERVICE_GEN)/* $(GAME_SERVICE_GEN)/*

# Phony targets
.PHONY: all generate clean

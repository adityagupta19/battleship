#!/usr/bin/env bash
# End-to-end smoke test (requires: docker compose up, websocat or wscat)
set -euo pipefail

GATEWAY="${GATEWAY:-ws://localhost:8080/ws}"

echo "=== Register users via grpcurl (optional) ==="
echo "grpcurl -plaintext -d '{\"username\":\"alice\"}' localhost:50051 userpb.UserService/RegisterUser"
echo "grpcurl -plaintext -d '{\"username\":\"bob\"}' localhost:50051 userpb.UserService/RegisterUser"

echo ""
echo "=== WebSocket flow ==="
echo "Terminal 1: websocat '${GATEWAY}?user_id=1'"
echo "Terminal 2: websocat '${GATEWAY}?user_id=2'"
echo ""
echo 'Send find_match: {"type":"find_match"}'
echo ""
echo 'Place ships (both players, example):'
cat <<'EOF'
{"type":"place_ships","game_id":1,"ships":[
  {"type":"carrier","start_x":0,"start_y":0,"horizontal":true},
  {"type":"battleship","start_x":2,"start_y":0,"horizontal":true},
  {"type":"cruiser","start_x":4,"start_y":0,"horizontal":true},
  {"type":"submarine","start_x":6,"start_y":0,"horizontal":true},
  {"type":"destroyer","start_x":8,"start_y":0,"horizontal":true}
]}
EOF
echo ""
echo 'Fire shot: {"type":"fire_shot","game_id":1,"x":0,"y":0}'
echo 'Get state: {"type":"get_state","game_id":1}'

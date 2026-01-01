#!/usr/bin/env bash
set -e

echo "Building shard_server..."
go build -o bin/shard_server ./cmd/shard_server

CONFIG_FILE="config.yml"
BINARY="./bin/shard_server"

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "config.yml not found"
  exit 1
fi

if [[ ! -x "$BINARY" ]]; then
  echo "shard_server binary not found at $BINARY"
  echo "Did you run: go build -o bin/shard_server ./cmd/shard_server?"
  exit 1
fi

SHARD_COUNT=$(grep '^shard_count:' "$CONFIG_FILE" | awk '{print $2}')

if [[ -z "$SHARD_COUNT" ]]; then
  echo "shard_count not found in config.yml"
  exit 1
fi

echo "Starting $SHARD_COUNT shard servers..."

PIDS=()

for ((i=0; i<SHARD_COUNT; i++)); do
  echo "Starting shard $i"
  $BINARY --shard-id=$i &
  PIDS+=($!)
done

echo "All shard servers started"
echo "PIDs: ${PIDS[*]}"

# Wait so script keeps control
wait

package server

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/go-redis/redismock/v9"
    "github.com/redis/go-redis/v9"
)

func newTestRedisServer(rdb *redis.Client) *BigTableLiteServer {
    return &BigTableLiteServer{
        redis: rdb,
    }
}

func newMockServer(t *testing.T) (*BigTableLiteServer, redismock.ClientMock) {
    db, mock := redismock.NewClientMock()

    return &BigTableLiteServer{
        redis: db,
    }, mock
}

func newLocalRedisServer(t *testing.T) *BigTableLiteServer {
    addr := os.Getenv("TEST_REDIS_ADDR")
    if addr == "" {
        addr = "localhost:6379"
    }

    rdb := redis.NewClient(&redis.Options{Addr: addr})

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    if err := rdb.Ping(ctx).Err(); err != nil {
        t.Skip("Redis not running locally — skipping integration test")
    }

    return &BigTableLiteServer{
        redis: rdb,
    }
}
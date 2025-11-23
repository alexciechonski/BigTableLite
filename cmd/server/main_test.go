package main

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/alexciechonski/BigTableLite/proto"
    "github.com/go-redis/redismock/v9"
    "github.com/redis/go-redis/v9"
)

// create mock redis for CI
func newMockServer(t *testing.T) (*BigTableLiteServer, redismock.ClientMock) {
    db, mock := redismock.NewClientMock()

    return &BigTableLiteServer{
        redisClient: db,
        useRedis:    true,
    }, mock
}

// create real redis for local dev
func newLocalRedisServer(t *testing.T) *BigTableLiteServer {
    addr := os.Getenv("TEST_REDIS_ADDR")
    if addr == "" {
        addr = "localhost:6379"
    }

    // try connecting
    rdb := redis.NewClient(&redis.Options{Addr: addr})
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    if err := rdb.Ping(ctx).Err(); err != nil {
        t.Skip("Redis not running locally — skipping integration tests")
    }

    return &BigTableLiteServer{
        redisClient: rdb,
        useRedis:    true,
    }
}

func TestGetEnv(t *testing.T) {
    os.Setenv("HELLO", "world")
    defer os.Unsetenv("HELLO")

    got := getEnv("HELLO", "fallback")
    if got != "world" {
        t.Fatalf("expected world, got %s", got)
    }
}

func TestSet(t *testing.T) {

    var server *BigTableLiteServer
    var mock redismock.ClientMock

    if os.Getenv("GITHUB_ACTIONS") == "true" {
        // GitHub Actions → no Redis → use mock
        server, mock = newMockServer(t)
    } else {
        // local machine → try real Redis
        server = newLocalRedisServer(t)
    }

    ctx := context.Background()

    t.Run("successful set", func(t *testing.T) {
        if mock != nil {
            mock.ExpectSet("test-key-1", "test-value-1", 0).SetVal("OK")
        }

        _, err := server.Set(ctx, &proto.SetRequest{
            Key:   "test-key-1",
            Value: "test-value-1",
        })

        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }

        if mock != nil {
            if err := mock.ExpectationsWereMet(); err != nil {
                t.Fatalf("unmet redis expectations: %v", err)
            }
        }
    })
}

func TestGet(t *testing.T) {

    var server *BigTableLiteServer
    var mock redismock.ClientMock

    if os.Getenv("GITHUB_ACTIONS") == "true" {
        server, mock = newMockServer(t)
    } else {
        server = newLocalRedisServer(t)
    }

    ctx := context.Background()

    t.Run("key exists", func(t *testing.T) {

        if mock != nil {
            mock.ExpectGet("hello").SetVal("world")
        }

        resp, _ := server.Get(ctx, &proto.GetRequest{
            Key: "hello",
        })

        if !resp.Found {
            t.Fatalf("expected key to be found")
        }
        if resp.Value != "world" {
            t.Fatalf("expected value 'world', got %s", resp.Value)
        }

        if mock != nil {
            if err := mock.ExpectationsWereMet(); err != nil {
                t.Fatalf("unmet redis expectations: %v", err)
            }
        }
    })
}

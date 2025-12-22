package server

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alexciechonski/BigTableLite/proto"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func newMockServer(t *testing.T) (*BigTableLiteServer, redismock.ClientMock) {
	db, mock := redismock.NewClientMock()

	return &BigTableLiteServer{
		redisClient: db,
		useRedis:    true,
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
		redisClient: rdb,
		useRedis:    true,
	}
}

func TestSet(t *testing.T) {
	var server *BigTableLiteServer
	var mock redismock.ClientMock

	if os.Getenv("GITHUB_ACTIONS") == "true" {
		server, mock = newMockServer(t)
	} else {
		server = newLocalRedisServer(t)
	}

	ctx := context.Background()

	if mock != nil {
		mock.ExpectSet("key1", "value1", 0).SetVal("OK")
	}

	_, err := server.Set(ctx, &proto.SetRequest{
		Key:   "key1",
		Value: "value1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet redis expectations: %v", err)
		}
	}
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

	if mock != nil {
		mock.ExpectGet("hello").SetVal("world")
	}

	resp, err := server.Get(ctx, &proto.GetRequest{Key: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Found {
		t.Fatalf("expected key to be found")
	}

	if resp.Value != "world" {
		t.Fatalf("expected 'world', got %s", resp.Value)
	}

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet redis expectations: %v", err)
		}
	}
}

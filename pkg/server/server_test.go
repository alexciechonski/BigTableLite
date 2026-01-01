package server

import (
    "context"
    "os"
    "testing"

    "github.com/alexciechonski/BigTableLite/proto"
    "github.com/go-redis/redismock/v9"
)

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
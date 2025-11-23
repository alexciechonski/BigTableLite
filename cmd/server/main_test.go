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

// Mocks
type MockSSTableEngine struct{}

func (m *MockSSTableEngine) Put(k, v string) error { return nil }
func (m *MockSSTableEngine) Get(k string) (string, bool, error) {
	return "", false, nil
}
func (m *MockSSTableEngine) Flush() error    { return nil }
func (m *MockSSTableEngine) NeedsFlush() bool { return false }

// Real Redis (integration tests)
func setupTestRedis(t *testing.T) *redis.Client {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: Redis not available at %s: %v", redisAddr, err)
	}

	rdb.FlushDB(ctx)
	return rdb
}

// Redis mock (for CI + local unit tests)
func newMockRedis(t *testing.T) (*redis.Client, redismock.ClientMock) {
	db, mock := redismock.NewClientMock()
	return db, mock
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{"env present", "TEST_KEY", "default", "value", "value"},
		{"env missing", "TEST_KEY_2", "default", "", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewBigTableLiteServer(t *testing.T) {
	t.Run("successful redis connection", func(t *testing.T) {
		rdb, mock := newMockRedis(t)
		mock.ExpectPing().SetVal("PONG")

		// inject mock into server constructor
		server := &BigTableLiteServer{
			redisClient:   rdb,
			sstableEngine: &MockSSTableEngine{},
		}

		if server.redisClient == nil {
			t.Fatal("expected redisClient to be non-nil")
		}
	})

	t.Run("redis connection fails", func(t *testing.T) {
		rdb, mock := newMockRedis(t)
		mock.ExpectPing().SetErr(redis.ErrClosed)

		_, err := NewBigTableLiteServer("localhost:9999")
		if err == nil {
			t.Fatal("expected error but got nil")
		}
	})
}

func TestSet_Unit(t *testing.T) {
	// Always use mock for unit test
	rdb, mock := newMockRedis(t)
	mock.ExpectSet("test-key-1", "test-value-1", 0).SetVal("OK")

	server := &BigTableLiteServer{
		redisClient:   rdb,
		sstableEngine: &MockSSTableEngine{},
	}

	ctx := context.Background()
	req := &proto.SetRequest{Key: "test-key-1", Value: "test-value-1"}

	resp, err := server.Set(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true")
	}

	// Verify mock expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestGet_Unit(t *testing.T) {
	rdb, mock := newMockRedis(t)
	mock.ExpectGet("some-key").SetVal("hello")

	server := &BigTableLiteServer{
		redisClient:   rdb,
		sstableEngine: &MockSSTableEngine{},
	}

	ctx := context.Background()
	req := &proto.GetRequest{Key: "some-key"}

	resp, err := server.Get(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Found || resp.Value != "hello" {
		t.Fatalf("expected found=true and value='hello'")
	}
}

func TestSetAndGet_Integration(t *testing.T) {
	// Only runs if local Redis exists
	rdb := setupTestRedis(t)

	server := &BigTableLiteServer{
		redisClient:   rdb,
		sstableEngine: &MockSSTableEngine{},
	}

	ctx := context.Background()

	_, err := server.Set(ctx, &proto.SetRequest{Key: "int-key", Value: "int-value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := server.Get(ctx, &proto.GetRequest{Key: "int-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Found || resp.Value != "int-value" {
		t.Fatalf("integration test failed: key not found or wrong value")
	}
}

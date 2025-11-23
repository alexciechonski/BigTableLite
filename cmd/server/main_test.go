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

// setupTestRedis creates a test Redis client
// In a real scenario, you might want to use testcontainers or a dedicated test Redis instance
func setupTestRedis(t *testing.T) *redis.Client {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping test: Redis not available at %s: %v", redisAddr, err)
	}

	// Clean up test data
	rdb.FlushDB(ctx)

	return rdb
}

func MockRedis(t *testing.T) (*redis.Client, redismock.ClientMock) {
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
		{
			name:         "returns environment variable when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default when environment variable not set",
			key:          "TEST_KEY_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNewBigTableLiteServer(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		redisAddr := os.Getenv("TEST_REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		}

		server, err := NewBigTableLiteServer(redisAddr)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if server == nil {
			t.Fatal("expected server to be non-nil")
		}
		if server.redisClient == nil {
			t.Fatal("expected redisClient to be non-nil")
		}
	})

	t.Run("connection failure", func(t *testing.T) {
		server, err := NewBigTableLiteServer("localhost:9999")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if server != nil {
			t.Error("expected server to be nil on error")
		}
		if err != nil && err.Error() == "" {
			t.Error("expected error message to contain 'failed to connect to Redis'")
		}
	})
}

func TestSet(t *testing.T) {
	var (
        rdb    *redis.Client
        mock   redismock.ClientMock
    )

    if os.Getenv("GITHUB_ACTIONS") == "true" {
        // CI → use mock
        rdb, mock = MockRedis(t)

        // define expected Redis behavior for the test
        mock.ExpectSet("test-key-1", "test-value-1", 0).SetVal("OK")

    } else {
        // Local dev → use real Redis if running
        rdb = setupTestRedis(t)
    }

    server := &BigTableLiteServer{
        redisClient: rdb,
    }

	ctx := context.Background()

	t.Run("successful set", func(t *testing.T) {
		req := &proto.SetRequest{
			Key:   "test-key-1",
			Value: "test-value-1",
		}

		resp, err := server.Set(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true, got false")
		}
		if resp.Message == "" {
			t.Error("expected non-empty message")
		}

		// Verify value was actually set in Redis
		val, err := rdb.Get(ctx, "test-key-1").Result()
		if err != nil {
			t.Fatalf("expected no error getting from Redis, got %v", err)
		}
		if val != "test-value-1" {
			t.Errorf("expected value %q, got %q", "test-value-1", val)
		}
	})

	t.Run("set overwrites existing key", func(t *testing.T) {
		key := "test-key-2"
		// Set initial value
		rdb.Set(ctx, key, "initial-value", 0)

		req := &proto.SetRequest{
			Key:   key,
			Value: "new-value",
		}

		resp, err := server.Set(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true, got false")
		}

		// Verify new value
		val, err := rdb.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("expected no error getting from Redis, got %v", err)
		}
		if val != "new-value" {
			t.Errorf("expected value %q, got %q", "new-value", val)
		}
	})

	t.Run("set with empty key", func(t *testing.T) {
		req := &proto.SetRequest{
			Key:   "",
			Value: "value",
		}

		resp, err := server.Set(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Redis allows empty keys, so this should succeed
		if !resp.Success {
			t.Errorf("expected success=true, got false")
		}
	})

	t.Run("set with empty value", func(t *testing.T) {
		req := &proto.SetRequest{
			Key:   "test-key-empty-value",
			Value: "",
		}

		resp, err := server.Set(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success=true, got false")
		}

		// Verify empty value was set
		val, err := rdb.Get(ctx, "test-key-empty-value").Result()
		if err != nil {
			t.Fatalf("expected no error getting from Redis, got %v", err)
		}
		if val != "" {
			t.Errorf("expected empty value, got %q", val)
		}
	})
}

func TestGet(t *testing.T) {
	rdb := setupTestRedis(t)
	server := &BigTableLiteServer{
		redisClient: rdb,
	}

	ctx := context.Background()

	t.Run("successful get existing key", func(t *testing.T) {
		key := "test-key-get-1"
		value := "test-value-get-1"
		rdb.Set(ctx, key, value, 0)

		req := &proto.GetRequest{
			Key: key,
		}

		resp, err := server.Get(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !resp.Found {
			t.Errorf("expected found=true, got false")
		}
		if resp.Value != value {
			t.Errorf("expected value %q, got %q", value, resp.Value)
		}
		if resp.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		req := &proto.GetRequest{
			Key: "non-existent-key",
		}

		resp, err := server.Get(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Found {
			t.Errorf("expected found=false, got true")
		}
		if resp.Value != "" {
			t.Errorf("expected empty value, got %q", resp.Value)
		}
		if resp.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("get with empty key", func(t *testing.T) {
		req := &proto.GetRequest{
			Key: "",
		}

		resp, err := server.Get(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Empty key doesn't exist unless explicitly set
		if resp.Found {
			t.Errorf("expected found=false, got true")
		}
	})

	t.Run("get after set", func(t *testing.T) {
		key := "test-key-get-after-set"
		value := "test-value-get-after-set"

		// Set the value
		setReq := &proto.SetRequest{
			Key:   key,
			Value: value,
		}
		setResp, err := server.Set(ctx, setReq)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !setResp.Success {
			t.Errorf("expected success=true, got false")
		}

		// Get the value
		getReq := &proto.GetRequest{
			Key: key,
		}
		getResp, err := server.Get(ctx, getReq)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !getResp.Found {
			t.Errorf("expected found=true, got false")
		}
		if getResp.Value != value {
			t.Errorf("expected value %q, got %q", value, getResp.Value)
		}
	})
}

func TestSetAndGetIntegration(t *testing.T) {
	rdb := setupTestRedis(t)
	server := &BigTableLiteServer{
		redisClient: rdb,
	}

	ctx := context.Background()

	t.Run("set then get workflow", func(t *testing.T) {
		key := "integration-test-key"
		value := "integration-test-value"

		// Set
		setResp, err := server.Set(ctx, &proto.SetRequest{
			Key:   key,
			Value: value,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !setResp.Success {
			t.Errorf("expected success=true, got false")
		}

		// Get
		getResp, err := server.Get(ctx, &proto.GetRequest{
			Key: key,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !getResp.Found {
			t.Errorf("expected found=true, got false")
		}
		if getResp.Value != value {
			t.Errorf("expected value %q, got %q", value, getResp.Value)
		}
	})

	t.Run("multiple sets and gets", func(t *testing.T) {
		testCases := []struct {
			key   string
			value string
		}{
			{"key1", "value1"},
			{"key2", "value2"},
			{"key3", "value3"},
		}

		// Set all
		for _, tc := range testCases {
			resp, err := server.Set(ctx, &proto.SetRequest{
				Key:   tc.key,
				Value: tc.value,
			})
			if err != nil {
				t.Fatalf("expected no error setting %q, got %v", tc.key, err)
			}
			if !resp.Success {
				t.Errorf("expected success=true for key %q, got false", tc.key)
			}
		}

		// Get all
		for _, tc := range testCases {
			resp, err := server.Get(ctx, &proto.GetRequest{
				Key: tc.key,
			})
			if err != nil {
				t.Fatalf("expected no error getting %q, got %v", tc.key, err)
			}
			if !resp.Found {
				t.Errorf("expected found=true for key %q, got false", tc.key)
			}
			if resp.Value != tc.value {
				t.Errorf("expected value %q for key %q, got %q", tc.value, tc.key, resp.Value)
			}
		}
	})
}

package server

import (
    "context"
    "os"
    "testing"

    "github.com/alexciechonski/BigTableLite/proto"
    "github.com/go-redis/redismock/v9"
    "github.com/redis/go-redis/v9"
)

func newRedisForBenchmark() (*redis.Client, redismock.ClientMock) {
    if os.Getenv("GITHUB_ACTIONS") == "true" {
        return redismock.NewClientMock()
    }

    return redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    }), nil
}

func BenchmarkServerRedisSet(b *testing.B) {
    ctx := context.Background()
    rdb, mock := newRedisForBenchmark()

    if mock != nil {
        mock.ExpectSet("bench-key", "bench-value", 0).SetVal("OK")
    }

    server := &BigTableLiteServer{
        redis: rdb, // Changed from redisClient to redis
    }

    req := &proto.SetRequest{
        Key:   "bench-key",
        Value: "bench-value",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        server.Set(ctx, req)
    }
}

func BenchmarkRedisDirectSet(b *testing.B) {
    ctx := context.Background()
    rdb, mock := newRedisForBenchmark()

    if mock != nil {
        mock.ExpectSet("bench-key", "bench-value", 0).SetVal("OK")
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rdb.Set(ctx, "bench-key", "bench-value", 0)
    }
}
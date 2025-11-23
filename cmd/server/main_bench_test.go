package main

import (
    "context"
    "testing"

    "github.com/redis/go-redis/v9"
    "github.com/alexciechonski/BigTableLite/proto"
)

func BenchmarkRedisSet(b *testing.B) {
    ctx := context.Background()

    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    server := &BigTableLiteServer{
        redisClient: rdb,
        useRedis:    true,
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

    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        rdb.Set(ctx, "bench-key", "bench-value", 0)
    }
}

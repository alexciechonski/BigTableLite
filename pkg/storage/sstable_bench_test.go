package storage

import (
    "fmt"
    "os"
    "strings"
    "testing"
)

func BenchmarkSSTablePut(b *testing.B) {
    engine, _ := NewSSTableEngine("./benchdata")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = engine.Put("bench-key", "bench-value")
    }
}

func BenchmarkSSTableGetExisting(b *testing.B) {
    engine, _ := NewSSTableEngine("./benchdata")
    _ = engine.Put("bench-key", "bench-value")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        engine.Get("bench-key")
    }
}

func BenchmarkSSTableGetMissing(b *testing.B) {
    engine, _ := NewSSTableEngine("./benchdata")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        engine.Get("key-does-not-exist")
    }
}

func BenchmarkSSTablePutSequential(b *testing.B) {
    engine, _ := NewSSTableEngine("./benchdata")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        key := fmt.Sprintf("key-%d", i)
        _ = engine.Put(key, "value")
    }
}

func BenchmarkSSTablePutOverwrite(b *testing.B) {
    engine, _ := NewSSTableEngine("./benchdata")
    _ = engine.Put("bench-key", "initial")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = engine.Put("bench-key", "new-value")
    }
}

func BenchmarkSSTableFlush(b *testing.B) {
    engine, _ := NewSSTableEngine("./benchdata")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    for i := 0; i < 10000; i++ {
        engine.Put(fmt.Sprintf("k-%d", i), "v")
    }

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        engine.Flush()
    }
}

func BenchmarkSSTablePutLargeValue(b *testing.B) {
    large := strings.Repeat("x", 1024*100) // 100 KB
    engine, _ := NewSSTableEngine("./benchdata")
    b.Cleanup(func() { os.RemoveAll("./benchdata") })

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = engine.Put("large-key", large)
    }
}

package storage

func BenchmarkSSTablePut(b *testing.B) {
    engine, _ := storage.NewSSTableEngine("./benchdata")

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        engine.Put("bench-key", "bench-value")
    }
}
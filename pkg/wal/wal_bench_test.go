package wal

import (
	"testing"
)

func BenchmarkSerializeOperation(b *testing.B) {
	key := []byte("my-key")
	value := []byte("my-value-1234567890")

	for i := 0; i < b.N; i++ {
		_, err := SerializeOperation("set", key, value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

package wal

import (
	"testing"
	"os"
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

func BenchmarkWalAppend(b *testing.B) {
	tmpfile := "wal_bench.txt"
	os.Remove(tmpfile)
	w, err := NewWal(tmpfile)
	if err != nil {
		b.Fatal(err)
	}
	defer w.Close()

	entry, _ := SerializeOperation("set", []byte("k"), []byte("value"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := w.Append(entry); err != nil {
			b.Fatal(err)
		}
	}
}

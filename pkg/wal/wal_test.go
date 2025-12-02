package wal

import (
    "bytes"
    "os"
    "testing"
)

func TestWrite(t *testing.T) {
    testFile := "test_wal.txt"
    defer os.Remove(testFile)

    // Create WAL instance
    wal, err := NewWal(testFile)
    if err != nil {
        t.Fatalf("Failed to create WAL: %v", err)
    }

    // Close before reading file later
    defer wal.Close()

    // test serialization
    key := []byte("key1")
    value := []byte("value1")

    entry, err := SerializeOperation("set", key, value)
    if err != nil {
        t.Fatalf("SerializeOperation failed: %v", err)
    }

    if len(entry) == 0 {
        t.Fatal("SerializeOperation returned empty entry")
    }

    // Run append
    if err := wal.Append(entry); err != nil {
        t.Fatalf("Append failed: %v", err)
    }

    // Read file contents
    wal.Close()
    data, err := os.ReadFile(testFile)
    if err != nil {
        t.Fatalf("Failed to read WAL file: %v", err)
    }

    if len(data) == 0 {
        t.Fatal("WAL file is empty after append")
    }

    // Compare file bytes to original entry
    if !bytes.Equal(data, entry) {
        t.Errorf("WAL file contents do not match entry.\nExpected: %x\nGot:      %x", entry, data)
    }

    // Step 5: Deserialize and verify
    op, outKey, outValue, err := DeserializeOperation(data)
    if err != nil {
        t.Fatalf("DeserializeOperation failed: %v", err)
    }

    if op != "set" {
        t.Errorf("Expected op 'set', got %q", op)
    }

    if !bytes.Equal(outKey, key) {
        t.Errorf("Expected key %q, got %q", key, outKey)
    }

    if !bytes.Equal(outValue, value) {
        t.Errorf("Expected value %q, got %q", value, outValue)
    }
}

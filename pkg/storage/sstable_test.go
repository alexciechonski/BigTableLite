package storage

import (
	"os"
	"path/filepath"
	"testing"
	"fmt"
)

func setupTestEngine(t *testing.T) *SSTableEngine {
	// Create a temporary directory for test data
	testDir := filepath.Join(os.TempDir(), "bigtablelite_test", t.Name())
	os.RemoveAll(testDir) // Clean up any previous test data
	os.MkdirAll(testDir, 0755)
	testWALFile := filepath.Join(testDir, "wal.txt")

	engine, err := NewSSTableEngine(testDir, testWALFile)
	if err != nil {
		t.Fatalf("Failed to create SSTable engine: %v", err)
	}

	return engine
}

func cleanupTestEngine(t *testing.T, engine *SSTableEngine) {
	// Cleanup is handled by removing the test directory
	testDir := filepath.Join(os.TempDir(), "bigtablelite_test", t.Name())
	os.RemoveAll(testDir)
}

func TestSSTableEngine_PutAndGet(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	// Test basic put and get
	err := engine.Put("key1", "value1")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, found, err := engine.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected key to be found")
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", value)
	}
}

func TestSSTableEngine_GetNonExistent(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	value, found, err := engine.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Expected key to not be found")
	}
	if value != "" {
		t.Errorf("Expected empty value, got '%s'", value)
	}
}

func TestSSTableEngine_Overwrite(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	// Put initial value
	err := engine.Put("key1", "value1")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Overwrite with new value
	err = engine.Put("key1", "value2")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify new value
	value, found, err := engine.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected key to be found")
	}
	if value != "value2" {
		t.Errorf("Expected value 'value2', got '%s'", value)
	}
}

func TestSSTableEngine_MultipleKeys(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	// Put multiple keys
	testCases := []struct {
		key   string
		value string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	for _, tc := range testCases {
		err := engine.Put(tc.key, tc.value)
		if err != nil {
			t.Fatalf("Put failed for key %s: %v", tc.key, err)
		}
	}

	// Get all keys
	for _, tc := range testCases {
		value, found, err := engine.Get(tc.key)
		if err != nil {
			t.Fatalf("Get failed for key %s: %v", tc.key, err)
		}
		if !found {
			t.Errorf("Expected key %s to be found", tc.key)
		}
		if value != tc.value {
			t.Errorf("Expected value '%s' for key %s, got '%s'", tc.value, tc.key, value)
		}
	}
}

func TestSSTableEngine_Flush(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	// Put some data
	err := engine.Put("key1", "value1")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Manually flush
	err = engine.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify data is still accessible after flush
	value, found, err := engine.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected key to be found after flush")
	}
	if value != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", value)
	}
}

func TestSSTableEngine_EmptyKey(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	err := engine.Put("", "empty_key_value")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, found, err := engine.Get("")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected empty key to be found")
	}
	if value != "empty_key_value" {
		t.Errorf("Expected value 'empty_key_value', got '%s'", value)
	}
}

func TestSSTableEngine_EmptyValue(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	err := engine.Put("key1", "")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, found, err := engine.Get("key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected key to be found")
	}
	if value != "" {
		t.Errorf("Expected empty value, got '%s'", value)
	}
}

func TestSSTableEngine_LargeValue(t *testing.T) {
	engine := setupTestEngine(t)
	defer cleanupTestEngine(t, engine)

	// Create a large value (1KB) without null bytes
	largeValue := make([]byte, 1024)
	for i := range largeValue {
		largeValue[i] = byte((i % 255) + 1) // Avoid null bytes (0x00)
	}

	err := engine.Put("large_key", string(largeValue))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, found, err := engine.Get("large_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected key to be found")
	}
	if len(value) != len(largeValue) {
		t.Errorf("Expected value length %d, got %d", len(largeValue), len(value))
	}
}

func TestSStableEngine_Replay(t *testing.T) {

	// Setup Test engine
	baseDir := filepath.Join(os.TempDir(), t.Name())
	os.RemoveAll(baseDir)
    testDataDir := filepath.Join(baseDir, "data")
    os.MkdirAll(testDataDir, 0755)
	testWALFile := filepath.Join(baseDir, "wal.txt")

	engine1, err := NewSSTableEngine(testDataDir, testWALFile)
	if err != nil {
		t.Fatalf("Failed to create SSTable engine: %v", err)
	}

	// Add entries
	if err := engine1.Put("a", "1"); err != nil {
        t.Fatalf("Put(a) failed: %v", err)
    }
    if err := engine1.Put("b", "2"); err != nil {
        t.Fatalf("Put(b) failed: %v", err)
    }
    if err := engine1.Put("c", "3"); err != nil {
        t.Fatalf("Put(c) failed: %v", err)
    }
	
	// truncate last write
	walPath := engine1.wal.Path()
	engine1.wal.Close()
    info, err := os.Stat(walPath)
    if err != nil {
        t.Fatalf("Failed to stat WAL file: %v", err)
    }
    size := info.Size()
    if err := os.Truncate(walPath, size-5); err != nil {
        t.Fatalf("Failed to truncate WAL: %v", err)
    }

	// Simulate crash
	engine1.DestroySSTableEngine()
	engine1 = nil
	engine2, err := NewSSTableEngine(testDataDir, testWALFile)
	if err != nil {
		t.Fatalf("Failed to recover SSTable engine: %v", err)
	}

	// Verify recovery
	val, found, err := engine2.Get("a")
	if err != nil { t.Fatal(err) }
	if !found { t.Errorf("expected a to be found") }
	if val != "1" {
		t.Errorf("Expected key 'a' to have value '1', got '%s'", val)
	}

	val, found, err = engine2.Get("b")
	if err != nil { t.Fatal(err) }
	if !found { t.Errorf("expected b to be found") }
	if val != "2" {
		t.Errorf("Expected key 'b' to have value '2', got '%s'", val)
	}

	val, found, err = engine2.Get("c")
	if err != nil { t.Fatal(err) }
	if found { 
		fmt.Println("val:", val)
		t.Errorf("expected c to be NOT found") 
	}

	// manual cleanup
	cleanupTestEngine(t, engine2)
}




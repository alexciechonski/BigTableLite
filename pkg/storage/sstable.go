package storage

/*
#cgo CXXFLAGS: -std=c++11 -I${SRCDIR}/../../sstable
#cgo LDFLAGS: -L${SRCDIR}/../../sstable -lsstable -lstdc++
#include "../../sstable/sstable.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
	"github.com/alexciechonski/BigTableLite/pkg/wal"
	"os"
	"fmt"
)

// SSTableEngine wraps the C++ SSTable implementation
type SSTableEngine struct {
	initialized bool
	wal *wal.WriteAheadLog
}

// NewSSTableEngine creates a new SSTable engine instance
func NewSSTableEngine(dataDir string) (*SSTableEngine, error) {
	engine := &SSTableEngine{}

	// Initialize WAL
	walPath := os.Getenv("WAL_PATH")
    if walPath == "" {
        walPath = "wal.txt" // default path
    }

    w, err := wal.NewWal(walPath)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize WAL: %w", err)
    }
    engine.wal = w

	// Initialize SSTable system (C++)
	cDataDir := C.CString(dataDir)
	defer C.free(unsafe.Pointer(cDataDir))
	
	success := C.sstable_init(cDataDir)
	if !success {
		return nil, errors.New("failed to initialize SSTable engine")
	}
	
	engine.initialized = true
	return engine, nil

	// replay on startup
	err = engine.wal.Replay(func(entry []byte) error {
		_, _, op, key, value, err := wal.DeserializeOperation(entry)
		if err != nil {
			return err
		}

		if op == "set" {
			cKey := C.CString(string(key))
			cVal := C.CString(string(value))
			defer C.free(unsafe.Pointer(cKey))
			defer C.free(unsafe.Pointer(cVal))
			C.sstable_put(cKey, cVal)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("WAL replay failed: %w", err)
	}
}

// Put stores a key-value pair in the memtable
func (e *SSTableEngine) Put(key, value string) error {
	if !e.initialized {
		return errors.New("engine not initialized")
	}
	
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	// serialize entry and append to WAL
	serializedOp, err := wal.SerializeOperation("set", []byte(key), []byte(value))
	if err != nil {
		return fmt.Errorf("failed to serialize operation for WAL: %w", err)
	}

	if err := e.wal.Append(serializedOp); err != nil {
		return fmt.Errorf("failed to append to WAL: %w", err)
	}
	
	success := C.sstable_put(cKey, cValue)
	if !success {
		return errors.New("failed to put key-value pair")
	}
	
	// Check if memtable needs flushing
	if C.sstable_needs_flush() {
		if err := e.Flush(); err != nil {
			return err
		}
	}

	return nil
}

// Get retrieves a value by key (checks memtable first, then SSTables)
func (e *SSTableEngine) Get(key string) (string, bool, error) {
	if !e.initialized {
		return "", false, errors.New("engine not initialized")
	}
	
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	
	var bytes C.sstable_bytes
	success := C.sstable_get(cKey, &bytes)
	defer C.sstable_free_bytes(&bytes)
	
	if !success {
		return "", false, nil // Key not found
	}
	
	if bytes.data == nil {
		return "", false, nil
	}
	
	value := C.GoStringN(bytes.data, C.int(bytes.len))
	return value, true, nil
}

// Flush writes the memtable to disk as a new SSTable
func (e *SSTableEngine) Flush() error {
	if !e.initialized {
		return errors.New("engine not initialized")
	}
	
	// flush sstable
	success := C.sstable_flush()
	if !success {
		return errors.New("failed to flush memtable")
	}

	// Rotate WAL (critical!)
    if err := e.wal.Close(); err != nil {
        return fmt.Errorf("failed to close WAL: %w", err)
    }

    // Delete old WAL file
    if err := os.Remove(e.wal.Path()); err != nil {
        return fmt.Errorf("failed to delete WAL file: %w", err)
    }

    // Create new empty WAL
    newWal, err := wal.NewWal(e.wal.Path())
    if err != nil {
        return fmt.Errorf("failed to create new WAL: %w", err)
    }

	e.wal = newWal

	return nil
}

// NeedsFlush checks if the memtable needs to be flushed
func (e *SSTableEngine) NeedsFlush() bool {
	if !e.initialized {
		return false
	}
	
	return bool(C.sstable_needs_flush())
}


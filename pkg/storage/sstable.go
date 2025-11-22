package storage

/*
#cgo CXXFLAGS: -std=c++11 -I../../cpp
#cgo LDFLAGS: -L../../cpp -lsstable -lstdc++
#include "../../cpp/sstable.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

// SSTableEngine wraps the C++ SSTable implementation
type SSTableEngine struct {
	initialized bool
}

// NewSSTableEngine creates a new SSTable engine instance
func NewSSTableEngine(dataDir string) (*SSTableEngine, error) {
	engine := &SSTableEngine{}
	
	cDataDir := C.CString(dataDir)
	defer C.free(unsafe.Pointer(cDataDir))
	
	success := C.sstable_init(cDataDir)
	if !success {
		return nil, errors.New("failed to initialize SSTable engine")
	}
	
	engine.initialized = true
	return engine, nil
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
	
	success := C.sstable_flush()
	if !success {
		return errors.New("failed to flush memtable")
	}
	
	return nil
}

// NeedsFlush checks if the memtable needs to be flushed
func (e *SSTableEngine) NeedsFlush() bool {
	if !e.initialized {
		return false
	}
	
	return bool(C.sstable_needs_flush())
}


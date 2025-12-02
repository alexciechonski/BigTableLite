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
	"fmt"
	"os"
	"unsafe"
	"strings"

	"github.com/alexciechonski/BigTableLite/pkg/wal"
)

type SSTableEngine struct {
	initialized bool
	wal         *wal.WriteAheadLog
}

func NewSSTableEngine(dataDir, WALPath string) (*SSTableEngine, error) {
	// INIT SSTable (clears memtable)
    cDir := C.CString(dataDir)
    defer C.free(unsafe.Pointer(cDir))
    if !C.sstable_init(cDir) {
        return nil, errors.New("failed to initialize sstable")
    }

    // Open WAL
    w, err := wal.NewWal(WALPath)
    if err != nil {
        return nil, err
    }

    engine := &SSTableEngine{wal: w}

    // Replay WAL
    err = w.Replay(func(entry []byte) error {
        op, key, value, err := wal.DeserializeOperation(entry)
        if err != nil {
            return err
        }

        if op == "set" {
            cKey := C.CString(string(key))
            cVal := C.CString(string(value))
            C.sstable_put(cKey, cVal)
            C.free(unsafe.Pointer(cKey))
            C.free(unsafe.Pointer(cVal))
        }
        return nil
    })

    if err != nil {
        // checksum mismatch = safe to ignore
        if !strings.Contains(err.Error(), "checksum mismatch") {
            return nil, fmt.Errorf("WAL replay failed: %w", err)
        }
    }

    engine.initialized = true
    return engine, nil
}

func (e *SSTableEngine) DestroySSTableEngine() {
	if e == nil {
        return
    }
    C.sstable_destroy()
    if e.wal != nil {
        e.wal.Close()
        e.wal = nil
    }
    e.initialized = false
}

func (e *SSTableEngine) Put(key, value string) error {
	if !e.initialized {
		return errors.New("engine not initialized")
	}

	// Write to WAL FIRST
	entry, err := wal.SerializeOperation("set", []byte(key), []byte(value))
	if err != nil {
		return err
	}

	if err := e.wal.Append(entry); err != nil {
		return fmt.Errorf("cannot append to WAL: %w", err)
	}

	// Then apply to memtable
	cKey := C.CString(key)
	cVal := C.CString(value)
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cVal))

	if !C.sstable_put(cKey, cVal) {
		return errors.New("sstable_put failed")
	}

	// Check if flushing needed
	if C.sstable_needs_flush() {
		return e.Flush()
	}

	return nil
}

func (e *SSTableEngine) Flush() error {
	if !e.initialized {
		return errors.New("engine not initialized")
	}

	if !C.sstable_flush() {
		return errors.New("sstable_flush failed")
	}

	// WAL rotation
	if err := e.wal.Close(); err != nil {
		return err
	}

	if err := os.Remove(e.wal.Path()); err != nil {
		return err
	}

	newWal, err := wal.NewWal(e.wal.Path())
	if err != nil {
		return err
	}

	e.wal = newWal
	return nil
}

func (e *SSTableEngine) Get(key string) (string, bool, error) {
	if !e.initialized {
		return "", false, errors.New("engine not initialized")
	}

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var bytes C.sstable_bytes
	ok := C.sstable_get(cKey, &bytes)
	defer C.sstable_free_bytes(&bytes)

	if !ok || bytes.data == nil {
		return "", false, nil
	}

	return C.GoStringN(bytes.data, C.int(bytes.len)), true, nil
}

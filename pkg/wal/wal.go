package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"time"
)

type WriteAheadLog struct {
	path   string
	file   *os.File
	mu     sync.Mutex
	stopCh chan struct{}
}

func NewWal(path string) (*WriteAheadLog, error) {
	wal := &WriteAheadLog{
		path:   path,
		stopCh: make(chan struct{}),
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	wal.file = f

	go wal.backgroundSync(100 * time.Millisecond)

	return wal, nil
}

func (wal *WriteAheadLog) Close() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	if wal.file == nil {
		return nil
	}

	close(wal.stopCh)
	_ = wal.file.Sync()
	err := wal.file.Close()
	wal.file = nil
	return err
}

func SerializeOperation(operation string, key, value []byte) ([]byte, error) {
	var opType byte

	switch operation {
	case "set":
		opType = 0x01
	case "delete":
		opType = 0x02
	default:
		return nil, fmt.Errorf("unknown operation %q", operation)
	}

	keyLen := uint32(len(key))
	valueLen := uint32(len(value))

	// build payload
	payload := make([]byte, 0, 1+4+4+len(key)+len(value))
	payload = append(payload, opType)

	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, keyLen)
	payload = append(payload, tmp...)

	binary.BigEndian.PutUint32(tmp, valueLen)
	payload = append(payload, tmp...)

	payload = append(payload, key...)
	payload = append(payload, value...)

	// compute checksum
	check := crc32.ChecksumIEEE(payload)

	// header = [recordLength][checksum]
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(header[4:8], check)

	// final entry
	return append(header, payload...), nil
}

func DeserializeOperation(entry []byte) (op string, key, value []byte, err error) {
	if len(entry) < 13 {
		return "", nil, nil, fmt.Errorf("entry too short")
	}

	recordLength := binary.BigEndian.Uint32(entry[0:4])
	check := binary.BigEndian.Uint32(entry[4:8])

	payload := entry[8:]

	if uint32(len(payload)) != recordLength {
		return "", nil, nil, fmt.Errorf("record length mismatch")
	}

	if crc32.ChecksumIEEE(payload) != check {
		return "", nil, nil, fmt.Errorf("checksum mismatch")
	}

	opType := payload[0]
	keyLen := binary.BigEndian.Uint32(payload[1:5])
	valLen := binary.BigEndian.Uint32(payload[5:9])

	if int(keyLen)+int(valLen)+9 != len(payload) {
		return "", nil, nil, fmt.Errorf("payload lengths inconsistent")
	}

	key = payload[9 : 9+keyLen]
	value = payload[9+keyLen : 9+keyLen+valLen]

	switch opType {
	case 0x01:
		op = "set"
	case 0x02:
		op = "delete"
	default:
		return "", nil, nil, fmt.Errorf("unknown op type")
	}

	return op, key, value, nil
}

func (wal *WriteAheadLog) Append(entry []byte) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	if wal.file == nil {
		return fmt.Errorf("WAL file is closed")
	}

	if _, err := wal.file.Write(entry); err != nil {
		return err
	}

	return wal.file.Sync() // ensures durability
}

func (wal *WriteAheadLog) Sync() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	if wal.file == nil {
		return nil
	}
	return wal.file.Sync()
}

func (wal *WriteAheadLog) Replay(fn func(entry []byte) error) error {
	f, err := os.Open(wal.path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		header := make([]byte, 8)
		_, err := io.ReadFull(f, header)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return nil // corrupted tail
		}

		recLen := binary.BigEndian.Uint32(header[:4])
		payload := make([]byte, recLen)

		_, err = io.ReadFull(f, payload)
		if err != nil {
			return nil // truncated entry, ignore tail
		}

		entry := append(header, payload...)

		if err := fn(entry); err != nil {
			return err
		}
	}
}

func (wal *WriteAheadLog) backgroundSync(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-wal.stopCh:
			return
		case <-t.C:
			wal.Sync()
		}
	}
}

func (wal *WriteAheadLog) Path() string {
	return wal.path
}

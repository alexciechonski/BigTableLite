package wal

import (
	"fmt"
	"encoding/binary"
	"hash/crc32"
	"os"
	"sync"
	"bufio"
)

type WriteAheadLog struct {
	path string
	file *os.File
    mu   sync.Mutex
}

func NewWal(path string) (*WriteAheadLog, error) {
	wal := &WriteAheadLog{
        path: path,
    }

    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return nil, err
    }

    wal.file = f
    return wal, nil
}

func (wal *WriteAheadLog) Close() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()
	if wal.file == nil {
		return nil
	}
	wal.file.Sync()
	wal.file.Close()
	wal.file = nil
	return nil
}

// Serialize operation into a byte slice for WAL
func SerializeOperation(operation string, key, value []byte) ([]byte, error){
	var recordLength, checkSum, operationType, keyLength, valueLength []byte

	// handle operation type
	if operation == "set" {
		operationType = []byte{0x01} // Set operation
	// } else if operation == "delete" {
	// 	operationType = []byte{0x02} // Delete operation
	} else {
		return nil, fmt.Errorf("unknown operation type: %s", operation)
	}

	// key and value lengths
	keyLength = make([]byte, 4)
	binary.BigEndian.PutUint32(keyLength, uint32(len(key)))

	valueLength = make([]byte, 4)
	binary.BigEndian.PutUint32(valueLength, uint32(len(value)))

	// get checksum
	payload := []byte{}
	payload = append(payload, operationType...)
	payload = append(payload, keyLength...)
	payload = append(payload, valueLength...)
	payload = append(payload, key...)
	payload = append(payload, value...)

	checkSumValue := crc32.ChecksumIEEE(payload)
	checkSum = make([]byte, 4)
	binary.BigEndian.PutUint32(checkSum, checkSumValue)

	// record length
	recordLength = make([]byte, 4)
	binary.BigEndian.PutUint32(recordLength, uint32(len(payload)))
	
	// final serialized entry
	entry, header := make([]byte, 0), make([]byte, 0)
	header = append(header, recordLength...)
	header = append(header, checkSum...)

	entry = append(entry, header...)
	entry = append(entry, payload...)
	return entry, nil
}

// unserialze WAL entry into operation, key, value
func DeserializeOperation(entry []byte) (string, []byte, []byte, error) {
	if len(entry) < 13 {
		return "", nil, nil, fmt.Errorf("entry too short")
	}

	// read record length and checksum
	recordLength := binary.BigEndian.Uint32(entry[0:4])
	checkSum := binary.BigEndian.Uint32(entry[4:8])

	payload := entry[8:]

	// read operation type
	operationType := payload[0]
	keyLength := binary.BigEndian.Uint32(payload[1:5])
	valueLength := binary.BigEndian.Uint32(payload[5:9])

	key := payload[9 : 9+keyLength]
	value := payload[9+keyLength : 9+keyLength+valueLength]

	var operation string
	if operationType == 0x01 {
		operation = "set"
	} else if operationType == 0x02 {
		operation = "delete"
	} else {
		return "", nil, nil, fmt.Errorf("unknown operation type: %d", operationType)
	}

	return recordLength, checkSum, operation, key, value, nil
}

// Append parsed entry to WAL file
func (wal *WriteAheadLog) Append(entry []byte) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	// append entry
	_, err := wal.file.Write(entry)
	if err != nil {
		return err
	}
	return nil
}

// write the WAL file from RAM to the physical disk.
func (wal *WriteAheadLog) Sync() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()
	if wal.file == nil {
		return fmt.Errorf("WAL file is not open")
	}
	return wal.file.Sync()
}

// Replay WAL entries by calling ProcessFunc for each entry
func (wal WriteAheadLog) Replay(ProcessFunc func(entry []byte) error) error {
	wal.mu.Lock()
    defer wal.mu.Unlock()

    f, err := os.Open(wal.path)
    if err != nil {
        return err
    }
    defer f.Close()

    for {
        // Read header: 8 bytes
        header := make([]byte, 8)
        _, err := io.ReadFull(f, header)

        if err == io.EOF {
            return nil
        }
        if err != nil {
            // partial header ->  corrupted tail -> stop replay quietly
            return nil
        }

        // Parse recordLength from header
        recordLength := binary.BigEndian.Uint32(header[0:4])

        // Read payload
        payload := make([]byte, recordLength)
        _, err = io.ReadFull(f, payload)

        if err != nil {
            // corrupted tail
            return nil
        }

        // Build full entry for your Deserialize
        entry := append(header, payload...)

        // Deserialize FULL record 
        _, _, operation, key, value, err := DeserializeOperation(entry)
        if err != nil {
            // corrupted record
            return nil
        }

        if err := ProcessFunc(operation, key, value); err != nil {
            return err
        }
    }
	
}

package wal

import (
	"fmt",
	"encoding/binary",
	"hash/crc32",
	"os",
	"sync",
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
		wal.mu.Unlock()
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
	} else if operation == "delete" {
		operationType = []byte{0x02} // Delete operation
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

// Append parsed entry to WAL file
func (wal WriteAheadLog) Append(entry []byte) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	// open file in append mode
	f, err := os.OpenFile(wal.path, os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// append entry
	_, err = f.Write(entry)
	if err != nil {
		return err
	}
	
	return nil

}

func (wal *WriteAheadLog) Sync() error {
	// write the WAL file from RAM to the physical disk.
	wal.mu.Lock()
	defer wal.mu.Unlock()
	if wal.file == nil {
		return fmt.Errorf("WAL file is not open")
	}
	return wal.file.Sync()
}

func (wal WriteAheadLog) Flush() error {
	// Flush WAL entries to remote storage
}

func (wal WriteAheadLog) Replay(ProcessFunc func(entry []byte) error) error {
	// Replay WAL entries by calling ProcessFunc for each entry
}

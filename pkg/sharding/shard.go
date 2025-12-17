package sharding

import (
	"fmt"
	"sync"

	"github.com/alexciechonski/BigTableLite/pkg/storage/sstable.go"
)

type Shard struct {
	Id int
	Address string
	Engine *SSTableEngine
	mu     sync.Mutex
}

func NewShard(id int, address, dataDir, WALPath string) (*Shard, error) {
	engine, err := storage.NewSSTableEngine(dataDir, walPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage engine: %w", err)
	}

	return &Shard{
		ID:      id,
		Address: address,
		engine:  engine,
	}, nil
}

func (s *Shard) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.engine != nil {
		s.engine.DestroySSTableEngine()
		s.engine = nil
	}

	return nil
}

func (s *Shard) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.engine == nil {
		return fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.engine.Put(key, value)
}

func (s *Shard) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.engine == nil {
		return fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.engine.Delete(key)
}

func (s *Shard) Get(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.engine == nil {
		return "", false, fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.engine.Get(key)
}

func (s *Shard) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.engine == nil {
		return fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.engine.Flush()
}

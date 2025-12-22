package sharding

import (
	"fmt"
	"sync"

	"github.com/alexciechonski/BigTableLite/pkg/storage"
)

type Shard struct {
	ID int
	Address string
	Engine *storage.SSTableEngine
	mu     sync.Mutex
}

func NewShard(id int, address, dataDir, WALPath string) (*Shard, error) {
	engine, err := storage.NewSSTableEngine(dataDir, WALPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage engine: %w", err)
	}

	return &Shard{
		ID:      id,
		Address: address,
		Engine:  engine,
	}, nil
}

func (s *Shard) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Engine != nil {
		s.Engine.DestroySSTableEngine()
		s.Engine = nil
	}

	return nil
}

func (s *Shard) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Engine == nil {
		return fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.Engine.Put(key, value)
}

func (s *Shard) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Engine == nil {
		return fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.Engine.Delete(key)
}

func (s *Shard) Get(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Engine == nil {
		return "", false, fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.Engine.Get(key)
}

func (s *Shard) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Engine == nil {
		return fmt.Errorf("shard %d is not initialized", s.ID)
	}

	return s.Engine.Flush()
}

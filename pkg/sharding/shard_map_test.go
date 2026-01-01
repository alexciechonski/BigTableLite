package sharding

import (
	"testing"

	"github.com/alexciechonski/BigTableLite/pkg/config"
)

func TestShardResolveDeterministic(t *testing.T) {
	shards := []config.ShardDescriptor{
		{ID: 0, Address: "localhost:5000"},
		{ID: 1, Address: "localhost:5001"},
		{ID: 2, Address: "localhost:5002"},
	}

	sm, err := NewShardMap(shards)
	if err != nil {
		t.Fatal(err)
	}

	s1 := sm.Resolve("user:123")
	s2 := sm.Resolve("user:123")

	if s1.ID != s2.ID {
		t.Fatalf("expected deterministic shard, got %d and %d", s1.ID, s2.ID)
	}
}

func TestShardResolveInRange(t *testing.T) {
	shards := []config.ShardDescriptor{
		{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3},
	}

	sm, err := NewShardMap(shards)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"a", "b", "c", "d", "e"} {
		shard := sm.Resolve(key)
		if shard.ID < 0 || shard.ID >= len(shards) {
			t.Fatalf("invalid shard id %d", shard.ID)
		}
	}
}

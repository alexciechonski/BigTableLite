package sharding 

import (
	"hash/fnv"
	"fmt"

	"github.com/alexciechonski/BigTableLite/pkg/sharding/shard.go"
)

type ShardMap struct {
	shards []Shard
}

func NewShardMap(shards []Shard) (*ShardMap, error) {
	if len(shards) == 0 {
		return nil, fmt.Errorf("shard map must contain at least one shard")
	}

	return &ShardMap{
		shards: shards,
	}, nil
}

func (sm *ShardMap) Resolve(key string) Shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))

	index := int(h.Sum32()) % len(sm.shards)
	return sm.shards[index]
}
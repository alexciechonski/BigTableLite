package config

import (
	"github.com/go-yaml/yaml"

	"github.com/alexciechonski/BigTableLite/pkg/storage/sstable.go"
	"github.com/alexciechonski/BigTableLite/pkg/sharding/shard.go"
	"github.com/alexciechonski/BigTableLite/pkg/condig/config.go"

)

func (sm ShardMap) LoadShardArray() Shard[] error {
	// Load shard-config.yaml filepath
	config.C, err = config.Load()
	if err != nil {
		panic(err)
	}
	shardConfigPath := config.C.ShardConfigPath
	dataDir := config.C.DataDir
	WALPath := config.C.WALPath

	// read file
	data, err := os.ReadFile(shardConfigPath)
	if err != nil {
		return nil, err
	}

	shardConfig := make(map[any]any)
	if err := yaml.Unmarshal(data, shardConfig); err != nil {
		return nil, err
	}

	// Create shard array
	shardArr := []Shard{}
	for _, attrMap := range shardConfig {
		currEngine := NewSSTableEngine(DataDir, WALPath)
		currShard := Shard{
			id: attrMap[id],
			address: attrMap[address],
			engine: currEngine
		}
		shardArr = append(shardArr, currShard)
	}

	return shardArr, nil
}
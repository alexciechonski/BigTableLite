package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ShardDescriptor struct {
	ID      int    `yaml:"id"`
	Address string `yaml:"address"`
}

type Cluster struct {
	Shards []ShardDescriptor
}

// helper type to match the YAML shape
type rawShard map[string][]map[string]interface{}

// LoadCluster loads shard metadata from a YAML file
func LoadCluster(path string) (*Cluster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cluster config: %w", err)
	}

	var raw rawShard
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal cluster config: %w", err)
	}

	cluster := &Cluster{}

	for _, entries := range raw {
		var sd ShardDescriptor

		for _, field := range entries {
			if v, ok := field["id"]; ok {
				sd.ID = v.(int)
			}
			if v, ok := field["address"]; ok {
				sd.Address = v.(string)
			}
		}

		cluster.Shards = append(cluster.Shards, sd)
	}

	return cluster, nil
}

// GetShardByID returns shard metadata for a given shard ID
func (c *Cluster) GetShardByID(id int) (ShardDescriptor, error) {
	for _, shard := range c.Shards {
		if shard.ID == id {
			return shard, nil
		}
	}
	return ShardDescriptor{}, fmt.Errorf("shard %d not found", id)
}

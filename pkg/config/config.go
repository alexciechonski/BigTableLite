package config

import (
	"os"
	"path/filepath"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WALPath     string `yaml:"wal_path"`
	DataDir     string `yaml:"data_dir"`
	GRPCPort    string `yaml:"grpc_port"`
	MetricsPort string `yaml:"metrics_port"`
	UseRedis    bool   `yaml:"use_redis"`
	RedisAddr   string `yaml:"redis_addr"`
	ShardCount int     `yaml: "shard_count"`
	ShardConfigPath string `yaml: "shard_config_path"`
}

func Load() (*Config, error) {

	fmt.Println("Loading Config...")

	cfgPath := os.Getenv("CONFIG_PATH")
    if cfgPath == "" {
        cfgPath = "./config.yml" 
    }

	fmt.Println(cfgPath)

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.ApplyEnv()

	return cfg, nil
}

// ApplyEnv overrides YAML values with environment variables if they exist.
func (c *Config) ApplyEnv() {
	override := func(env string, target *string) {
		if v, ok := os.LookupEnv(env); ok {
			*target = v
		}
	}

	override("WAL_PATH", &c.WALPath)
	override("DATA_DIR", &c.DataDir)
	override("GRPC_PORT", &c.GRPCPort)
	override("METRICS_PORT", &c.MetricsPort)
	override("REDIS_ADDR", &c.RedisAddr)

	if v, ok := os.LookupEnv("USE_REDIS"); ok {
		c.UseRedis = (v == "true" || v == "1")
	}
}

// ResolvePaths converts relative paths to absolute using project root.
func (c *Config) ResolvePaths(root string) {
	if !filepath.IsAbs(c.WALPath) {
		c.WALPath = filepath.Join(root, c.WALPath)
	}
	if !filepath.IsAbs(c.DataDir) {
		c.DataDir = filepath.Join(root, c.DataDir)
	}
}


var C *Config


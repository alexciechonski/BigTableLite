package config

import (
	"os"
	"strconv"
    "path/filepath"
    "fmt"

	"gopkg.in/yaml.v3"
)

type Config struct {
    WALPath         string `yaml:"wal_path"`
    DataDir         string `yaml:"data_dir"`
    GRPCPort        string `yaml:"grpc_port"`
    MetricsPort     string `yaml:"metrics_port"`
    UseRedis        bool   `yaml:"use_redis"`
    RedisAddr       string `yaml:"redis_addr"`
    ShardCount      int    `yaml:"shard_count"`
    ShardConfigPath string `yaml:"shard_config_path"`
}

func Load() (*Config, error) {

	cfgPath := os.Getenv("CONFIG_PATH")
    if cfgPath == "" {
        cfgPath = "./config.yml" 
    }

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

    if !filepath.IsAbs(cfg.ShardConfigPath) {
		configDir := filepath.Dir(cfgPath)
		cfg.ShardConfigPath = filepath.Join(configDir, cfg.ShardConfigPath)
	}

	cfg.ApplyEnv()

    fmt.Println("Final ShardConfigPath:", cfg.ShardConfigPath)

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
    override("SHARD_CONFIG_PATH", &c.ShardConfigPath)

    if v, ok := os.LookupEnv("SHARD_COUNT"); ok {
        if i, err := strconv.Atoi(v); err == nil {
            c.ShardCount = i
        }
    }

    if v, ok := os.LookupEnv("USE_REDIS"); ok {
        c.UseRedis = (v == "true" || v == "1")
    }
}

var C *Config


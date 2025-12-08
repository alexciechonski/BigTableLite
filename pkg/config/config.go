package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WALPath     string `yaml:"wal_path"`
	DataDir     string `yaml:"data_dir"`
	GRPCPort    string `yaml:"grpc_port"`
	MetricsPort string `yaml:"metrics_port"`
	UseRedis    bool   `yaml:"use_redis"`
	RedisAddr   string `yaml:"redis_addr"`
}

func Load() (*Config, error) {
	root := ProjectRoot()

	data, err := os.ReadFile("config.yml")
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.ApplyEnv()
	cfg.ResolvePaths(root)

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

// ProjectRoot returns the absolute path of the project root directory.
func ProjectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

var C *Config
// var err error

package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

type Config struct {
	Env string `yaml:"env"`
}

var defaultConfig = Config{
	Env: EnvDev,
}

func (c *Config) validate() error {
	if c.Env != EnvDev && c.Env != EnvTest && c.Env != EnvProd {
		return fmt.Errorf("invalid env: %q", c.Env)
	}
	return nil
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	cfg := defaultConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

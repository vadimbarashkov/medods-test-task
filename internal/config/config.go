package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

type Server struct {
	Port           int           `yaml:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

var defaultServer = Server{
	Port:           8080,
	ReadTimeout:    5 * time.Second,
	WriteTimeout:   10 * time.Second,
	IdleTimeout:    time.Minute,
	MaxHeaderBytes: 1 << 20,
}

func (s *Server) validate() error {
	if !(s.Port >= 1 && s.Port <= 65535) {
		return fmt.Errorf("invalid port: %d", s.Port)
	}
	return nil
}

type Config struct {
	Env    string `yaml:"env"`
	Server Server `yaml:"server"`
}

var defaultConfig = Config{
	Env:    EnvDev,
	Server: defaultServer,
}

func (c *Config) validate() error {
	var errs []error

	if c.Env != EnvDev && c.Env != EnvTest && c.Env != EnvProd {
		errs = append(errs, fmt.Errorf("invalid env: %q", c.Env))
	}
	if err := c.Server.validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid server: %w", err))
	}

	return errors.Join(errs...)
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

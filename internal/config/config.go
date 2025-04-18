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
	var errs []error

	if !(s.Port >= 1 && s.Port <= 65535) {
		errs = append(errs, fmt.Errorf("invalid port: %d", s.Port))
	}
	if s.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("read_timeout must be positive: %v", s.ReadTimeout))
	}
	if s.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf("write_timeout must be positive: %v", s.WriteTimeout))
	}
	if s.IdleTimeout <= 0 {
		errs = append(errs, fmt.Errorf("idle_timeout must be positive: %v", s.IdleTimeout))
	}
	if s.MaxHeaderBytes < 0 {
		errs = append(errs, fmt.Errorf("max_header_bytes cannot be negative: %d", s.MaxHeaderBytes))
	}

	return errors.Join(errs...)
}

type Postgres struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
}

var defaultPostgres = Postgres{
	Host: "localhost",
	Port: 5432,
}

func (p *Postgres) validate() error {
	var errs []error

	if p.User == "" {
		errs = append(errs, errors.New("missing user"))
	}
	if p.Password == "" {
		errs = append(errs, errors.New("missing password"))
	}
	if !(p.Port >= 1 && p.Port <= 65535) {
		errs = append(errs, fmt.Errorf("invalid port: %d", p.Port))
	}
	if p.Database == "" {
		errs = append(errs, errors.New("missing database"))
	}

	return errors.Join(errs...)
}

type Config struct {
	Env      string   `yaml:"env"`
	Server   Server   `yaml:"server"`
	Postgres Postgres `yaml:"postgres"`
}

var defaultConfig = Config{
	Env:      EnvDev,
	Server:   defaultServer,
	Postgres: defaultPostgres,
}

func (c *Config) validate() error {
	var errs []error

	if c.Env != EnvDev && c.Env != EnvTest && c.Env != EnvProd {
		errs = append(errs, fmt.Errorf("invalid env: %q", c.Env))
	}
	if err := c.Server.validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid server: %w", err))
	}
	if err := c.Postgres.validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid postgres: %w", err))
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

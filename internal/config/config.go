package config

import (
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

type Tokens struct {
	AccessSecret  string        `yaml:"access_secret" validate:"required"`
	AccessTTL     time.Duration `yaml:"access_ttl" validate:"gt=0"`
	RefreshSecret string        `yaml:"refresh_secret" validate:"required"`
	RefreshTTL    time.Duration `yaml:"refresh_ttl" validate:"gt=0"`
}

var defautlTokens = Tokens{
	AccessTTL:  15 * time.Minute,
	RefreshTTL: time.Hour,
}

type Server struct {
	Port           int           `yaml:"port" validate:"required,min=1,max=65535"`
	ReadTimeout    time.Duration `yaml:"read_timeout" validate:"gt=0"`
	WriteTimeout   time.Duration `yaml:"write_timeout" validate:"gt=0"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" validate:"gt=0"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" validate:"gte=0"`
}

var defaultServer = Server{
	Port:           8080,
	ReadTimeout:    5 * time.Second,
	WriteTimeout:   10 * time.Second,
	IdleTimeout:    time.Minute,
	MaxHeaderBytes: 1 << 20,
}

func (s *Server) Addr() string {
	return fmt.Sprintf(":%d", s.Port)
}

type Postgres struct {
	User              string        `yaml:"user" validate:"required"`
	Password          string        `yaml:"password" validate:"required"`
	Host              string        `yaml:"host" validate:"required"`
	Port              int           `yaml:"port" validate:"required,min=1,max=65535"`
	Database          string        `yaml:"database" validate:"required"`
	SSLMode           string        `yaml:"sslmode" validate:"required,oneof=disable require verify-ca verify-full"`
	MaxConns          int32         `yaml:"max_conns" validate:"gt=0"`
	MinConns          int32         `yaml:"min_conns" validate:"gte=0"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime" validate:"gt=0"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time" validate:"gt=0"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period" validate:"gt=0"`
	MigrationsPath    string        `yaml:"migrations_path" validate:"required"`
}

var defaultPostgres = Postgres{
	Host:              "localhost",
	Port:              5432,
	SSLMode:           "disable",
	MaxConns:          20,
	MinConns:          4,
	MaxConnLifetime:   time.Hour,
	MaxConnIdleTime:   30 * time.Minute,
	HealthCheckPeriod: time.Minute,
	MigrationsPath:    "./migraions",
}

func (p *Postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&"+
			"pool_max_conns=%d&pool_min_conns=%d&"+
			"pool_max_conn_lifetime=%v&pool_max_conn_idle_time=%v&"+
			"pool_health_check_period=%v",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.Database,
		p.SSLMode,
		p.MaxConns,
		p.MinConns,
		p.MaxConnLifetime,
		p.MaxConnIdleTime,
		p.HealthCheckPeriod,
	)
}

type Config struct {
	Env      string   `yaml:"env" validate:"required,oneof=dev test prod"`
	Tokens   Tokens   `yaml:"tokens" validate:"required"`
	Server   Server   `yaml:"server" validate:"required"`
	Postgres Postgres `yaml:"postgres" validate:"required"`
}

var defaultConfig = Config{
	Env:      EnvDev,
	Tokens:   defautlTokens,
	Server:   defaultServer,
	Postgres: defaultPostgres,
}

var validate = validator.New()

func (c *Config) validate() error {
	return validate.Struct(c)
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	cfg := defaultConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

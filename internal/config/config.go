package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is loaded from configs/config.yaml (path overridable via CONFIG_PATH).
// Only secrets are overridable via environment variables: DB_PASSWORD, JWT_SECRET,
// REDIS_PASSWORD — everything else lives in the yaml file.
type Config struct {
	HTTP  HTTPConfig  `yaml:"http"`
	DB    DBConfig    `yaml:"db"`
	Redis RedisConfig `yaml:"redis"`
	JWT   JWTConfig   `yaml:"jwt"`
	Log   LogConfig   `yaml:"log"`
}

type HTTPConfig struct {
	Addr            string   `yaml:"addr"`
	ReadTimeout     Duration `yaml:"read_timeout"`
	WriteTimeout    Duration `yaml:"write_timeout"`
	IdleTimeout     Duration `yaml:"idle_timeout"`
	ShutdownTimeout Duration `yaml:"shutdown_timeout"`
}

type DBConfig struct {
	Host            string   `yaml:"host"`
	Port            int      `yaml:"port"`
	User            string   `yaml:"user"`
	Password        string   `yaml:"password"`
	Name            string   `yaml:"name"`
	MaxOpenConns    int      `yaml:"max_open_conns"`
	MaxIdleConns    int      `yaml:"max_idle_conns"`
	ConnMaxLifetime Duration `yaml:"conn_max_lifetime"`
}

// DSN builds a go-sql-driver/mysql data source name with parseTime enabled,
// required for scanning MySQL DATETIME/TIMESTAMP columns into time.Time.
func (c DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", c.User, c.Password, c.Host, c.Port, c.Name)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Secret string   `yaml:"secret"`
	TTL    Duration `yaml:"ttl"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

// Path resolves the config file path: CONFIG_PATH env var if set, else the
// repo-relative default used by both local `go run` and the Docker image.
func Path() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return "configs/config.yaml"
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
}

func (c Config) validate() error {
	if c.JWT.Secret == "" {
		return errors.New("jwt.secret is empty (set JWT_SECRET env var)")
	}
	if c.DB.Name == "" {
		return errors.New("db.name is empty")
	}
	return nil
}

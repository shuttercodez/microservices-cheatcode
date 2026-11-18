package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the complete application configuration mapped from YAML and Environment Variables.
type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

type AppConfig struct {
	Name                   string `yaml:"name"`
	Env                    string `yaml:"env"`
	Port                   int    `yaml:"port"`
	ShutdownTimeoutSeconds int    `yaml:"shutdown_timeout_seconds"`
}

// ShutdownTimeout returns the configured graceful shutdown duration.
func (a AppConfig) ShutdownTimeout() time.Duration {
	if a.ShutdownTimeoutSeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(a.ShutdownTimeoutSeconds) * time.Second
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
}

// DSN builds the PostgreSQL connection string for GORM.
// If DB_DSN is explicitly provided by a Cloud Vault, it takes highest precedence.
func (d DatabaseConfig) DSN() string {
	if envDSN := os.Getenv("DB_DSN"); envDSN != "" {
		return envDSN
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		d.Host, d.User, d.Password, d.Name, d.Port, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr string `yaml:"addr"`
}

type RabbitMQConfig struct {
	URL string `yaml:"url"`
}

// Load reads the YAML configuration file from the given path and applies environment variable
// overrides (used when deploying with AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault).
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:                   "inventory-order-service",
			Env:                    "development",
			Port:                   8080,
			ShutdownTimeoutSeconds: 5,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "appuser",
			Password: "apppassword",
			Name:     "inventory_db",
			SSLMode:  "disable",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		RabbitMQ: RabbitMQConfig{
			URL: "amqp://guest:guest@localhost:5672/",
		},
	}
}

// applyEnvOverrides allows Cloud Secret Managers / Vaults to override YAML values via standard env vars.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("APP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.App.Port = port
		}
	}

	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.Database.SSLMode = v
	}

	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		cfg.RabbitMQ.URL = v
	}
}

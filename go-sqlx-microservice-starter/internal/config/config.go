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
	Auth     AuthConfig     `yaml:"auth"`
}

type AuthConfig struct {
	JWTSecret       string `yaml:"jwt_secret"`
	TokenTTLSeconds int    `yaml:"token_ttl_seconds"`
	EnableRegister  bool   `yaml:"enable_register"`
}

func (a AuthConfig) TokenTTL() time.Duration {
	if a.TokenTTLSeconds <= 0 {
		return 3600 * time.Second
	}
	return time.Duration(a.TokenTTLSeconds) * time.Second
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

// DSN builds the PostgreSQL connection string.
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
	Addr       string `yaml:"addr"`
	TTLSeconds int    `yaml:"ttl_seconds"`
}

// TTL returns the configured Redis cache expiration duration.
func (r RedisConfig) TTL() time.Duration {
	if r.TTLSeconds <= 0 {
		return 300 * time.Second
	}
	return time.Duration(r.TTLSeconds) * time.Second
}

type RabbitMQConfig struct {
	URL string `yaml:"url"`
}

// Load reads the YAML configuration file from the given path and applies environment variable
// overrides (used when deploying with AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	applyEnvOverrides(&cfg)
	return &cfg, nil
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
	if v := os.Getenv("REDIS_TTL_SECONDS"); v != "" {
		if ttl, err := strconv.Atoi(v); err == nil {
			cfg.Redis.TTLSeconds = ttl
		}
	}
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		cfg.RabbitMQ.URL = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("JWT_TTL_SECONDS"); v != "" {
		if ttl, err := strconv.Atoi(v); err == nil {
			cfg.Auth.TokenTTLSeconds = ttl
		}
	}
	if v := os.Getenv("AUTH_ENABLE_REGISTER"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.Auth.EnableRegister = enabled
		}
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_YAMLAndVaultEnvOverride(t *testing.T) {
	// 1. Create a temporary config.yaml for testing
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
app:
  name: "test-service"
  env: "staging"
  port: 9090
  shutdown_timeout_seconds: 10
database:
  host: "db.internal"
  port: 5432
  user: "yaml_user"
  password: "yaml_password"
  name: "orders_db"
  ssl_mode: "require"
redis:
  addr: "redis.internal:6379"
rabbitmq:
  url: "amqp://yaml:yaml@mq.internal:5672/"
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("failed to write temp yaml: %v", err)
	}

	t.Run("Loads values from config.yaml accurately", func(t *testing.T) {
		cfg, err := Load(yamlPath)
		if err != nil {
			t.Fatalf("unexpected error loading config: %v", err)
		}
		if cfg.App.Port != 9090 {
			t.Errorf("expected port 9090, got %d", cfg.App.Port)
		}
		if cfg.Database.User != "yaml_user" {
			t.Errorf("expected db user 'yaml_user', got %s", cfg.Database.User)
		}
	})

	t.Run("Cloud Vault environment variables override YAML secrets", func(t *testing.T) {
		// Simulate secrets injected by AWS Secrets Manager / Azure Key Vault / HashiCorp Vault
		t.Setenv("DB_PASSWORD", "vault_super_secret_pw")
		t.Setenv("REDIS_ADDR", "prod-redis.cache.windows.net:6380")

		cfg, err := Load(yamlPath)
		if err != nil {
			t.Fatalf("unexpected error loading config: %v", err)
		}

		if cfg.Database.Password != "vault_super_secret_pw" {
			t.Errorf("expected Vault password override, got %s", cfg.Database.Password)
		}
		if cfg.Redis.Addr != "prod-redis.cache.windows.net:6380" {
			t.Errorf("expected Vault redis address override, got %s", cfg.Redis.Addr)
		}
	})
}

use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::path::Path;
use std::time::Duration;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Config {
    pub app: AppConfig,
    pub database: DatabaseConfig,
    pub redis: RedisConfig,
    pub rabbitmq: RabbitMQConfig,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AppConfig {
    pub name: String,
    pub env: String,
    pub port: u16,
    pub shutdown_timeout_seconds: u64,
}

impl AppConfig {
    pub fn shutdown_timeout(&self) -> Duration {
        if self.shutdown_timeout_seconds == 0 {
            Duration::from_secs(5)
        } else {
            Duration::from_secs(self.shutdown_timeout_seconds)
        }
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub user: String,
    pub password: String,
    pub name: String,
    pub ssl_mode: String,
}

impl DatabaseConfig {
    /// Builds the PostgreSQL connection URI for SQLx.
    /// If `DB_DSN` is explicitly provided by a Cloud Vault, it takes highest precedence.
    pub fn dsn(&self) -> String {
        if let Ok(env_dsn) = env::var("DB_DSN") {
            if !env_dsn.is_empty() {
                return env_dsn;
            }
        }
        format!(
            "postgres://{}:{}@{}:{}/{}?sslmode={}",
            self.user, self.password, self.host, self.port, self.name, self.ssl_mode
        )
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RedisConfig {
    pub addr: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RabbitMQConfig {
    pub url: String,
}

impl Config {
    /// Loads the YAML configuration file from the given path and applies environment variable
    /// overrides (used when deploying with AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault).
    pub fn load<P: AsRef<Path>>(path: P) -> anyhow::Result<Self> {
        let data = fs::read_to_string(path.as_ref()).map_err(|e| {
            anyhow::anyhow!("failed to read config file {}: {}", path.as_ref().display(), e)
        })?;

        let mut cfg: Config = serde_yaml_ng::from_str(&data)
            .map_err(|e| anyhow::anyhow!("failed to parse yaml config: {}", e))?;

        cfg.apply_env_overrides();
        Ok(cfg)
    }

    pub fn apply_env_overrides(&mut self) {
        if let Ok(v) = env::var("APP_ENV") {
            if !v.is_empty() {
                self.app.env = v;
            }
        }
        if let Ok(v) = env::var("APP_PORT") {
            if let Ok(port) = v.parse::<u16>() {
                self.app.port = port;
            }
        }

        if let Ok(v) = env::var("DB_HOST") {
            if !v.is_empty() {
                self.database.host = v;
            }
        }
        if let Ok(v) = env::var("DB_PORT") {
            if let Ok(port) = v.parse::<u16>() {
                self.database.port = port;
            }
        }
        if let Ok(v) = env::var("DB_USER") {
            if !v.is_empty() {
                self.database.user = v;
            }
        }
        if let Ok(v) = env::var("DB_PASSWORD") {
            if !v.is_empty() {
                self.database.password = v;
            }
        }
        if let Ok(v) = env::var("DB_NAME") {
            if !v.is_empty() {
                self.database.name = v;
            }
        }
        if let Ok(v) = env::var("DB_SSLMODE") {
            if !v.is_empty() {
                self.database.ssl_mode = v;
            }
        }

        if let Ok(v) = env::var("REDIS_ADDR") {
            if !v.is_empty() {
                self.redis.addr = v;
            }
        }
        if let Ok(v) = env::var("RABBITMQ_URL") {
            if !v.is_empty() {
                self.rabbitmq.url = v;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_load_config_yaml_and_vault_env_override() {
        let yaml_content = r#"
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
"#;

        let mut tmp_file = NamedTempFile::new().expect("failed to create temp file");
        tmp_file
            .write_all(yaml_content.as_bytes())
            .expect("failed to write temp yaml");

        // 1. Loads values from config.yaml accurately
        let cfg = Config::load(tmp_file.path()).expect("unexpected error loading config");
        assert_eq!(cfg.app.port, 9090);
        assert_eq!(cfg.database.user, "yaml_user");

        // 2. Cloud Vault environment variables override YAML secrets
        env::set_var("DB_PASSWORD", "vault_super_secret_pw");
        env::set_var("REDIS_ADDR", "prod-redis.cache.windows.net:6380");

        let cfg_overridden =
            Config::load(tmp_file.path()).expect("unexpected error loading config");
        assert_eq!(cfg_overridden.database.password, "vault_super_secret_pw");
        assert_eq!(
            cfg_overridden.redis.addr,
            "prod-redis.cache.windows.net:6380"
        );

        env::remove_var("DB_PASSWORD");
        env::remove_var("REDIS_ADDR");
    }
}

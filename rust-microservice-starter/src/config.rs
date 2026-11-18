use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::path::Path;
use std::time::Duration;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Config {
    #[serde(default)]
    pub app: AppConfig,
    #[serde(default)]
    pub database: DatabaseConfig,
    #[serde(default)]
    pub redis: RedisConfig,
    #[serde(default)]
    pub rabbitmq: RabbitMQConfig,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AppConfig {
    #[serde(default = "default_app_name")]
    pub name: String,
    #[serde(default = "default_app_env")]
    pub env: String,
    #[serde(default = "default_app_port")]
    pub port: u16,
    #[serde(default = "default_shutdown_timeout")]
    pub shutdown_timeout_seconds: u64,
}

fn default_app_name() -> String {
    "inventory-order-service".to_string()
}
fn default_app_env() -> String {
    "development".to_string()
}
fn default_app_port() -> u16 {
    8080
}
fn default_shutdown_timeout() -> u64 {
    5
}

impl Default for AppConfig {
    fn default() -> Self {
        Self {
            name: default_app_name(),
            env: default_app_env(),
            port: default_app_port(),
            shutdown_timeout_seconds: default_shutdown_timeout(),
        }
    }
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
    #[serde(default = "default_db_host")]
    pub host: String,
    #[serde(default = "default_db_port")]
    pub port: u16,
    #[serde(default = "default_db_user")]
    pub user: String,
    #[serde(default = "default_db_password")]
    pub password: String,
    #[serde(default = "default_db_name")]
    pub name: String,
    #[serde(default = "default_db_ssl_mode")]
    pub ssl_mode: String,
}

fn default_db_host() -> String {
    "localhost".to_string()
}
fn default_db_port() -> u16 {
    5432
}
fn default_db_user() -> String {
    "appuser".to_string()
}
fn default_db_password() -> String {
    "apppassword".to_string()
}
fn default_db_name() -> String {
    "inventory_db".to_string()
}
fn default_db_ssl_mode() -> String {
    "disable".to_string()
}

impl Default for DatabaseConfig {
    fn default() -> Self {
        Self {
            host: default_db_host(),
            port: default_db_port(),
            user: default_db_user(),
            password: default_db_password(),
            name: default_db_name(),
            ssl_mode: default_db_ssl_mode(),
        }
    }
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
    #[serde(default = "default_redis_addr")]
    pub addr: String,
}

fn default_redis_addr() -> String {
    "localhost:6379".to_string()
}

impl Default for RedisConfig {
    fn default() -> Self {
        Self {
            addr: default_redis_addr(),
        }
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RabbitMQConfig {
    #[serde(default = "default_rabbitmq_url")]
    pub url: String,
}

fn default_rabbitmq_url() -> String {
    "amqp://guest:guest@localhost:5672/".to_string()
}

impl Default for RabbitMQConfig {
    fn default() -> Self {
        Self {
            url: default_rabbitmq_url(),
        }
    }
}

impl Default for Config {
    fn default() -> Self {
        Self {
            app: AppConfig::default(),
            database: DatabaseConfig::default(),
            redis: RedisConfig::default(),
            rabbitmq: RabbitMQConfig::default(),
        }
    }
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

        let cfg_overridden = Config::load(tmp_file.path()).expect("unexpected error loading config");
        assert_eq!(cfg_overridden.database.password, "vault_super_secret_pw");
        assert_eq!(
            cfg_overridden.redis.addr,
            "prod-redis.cache.windows.net:6380"
        );

        env::remove_var("DB_PASSWORD");
        env::remove_var("REDIS_ADDR");
    }
}

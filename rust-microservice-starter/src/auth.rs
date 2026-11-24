use crate::config::AuthConfig;
use axum::body::Bytes;
use axum::extract::rejection::JsonRejection;
use axum::extract::{Request, State};
use axum::http::{header, HeaderMap, StatusCode};
use axum::middleware::Next;
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use base64::engine::general_purpose::{STANDARD, URL_SAFE_NO_PAD};
use base64::Engine;
use chrono::{DateTime, Utc};
use hmac::{Hmac, Mac};
use serde::{Deserialize, Serialize};
use serde_json::json;
use sha2::Sha256;
use sqlx::PgPool;
use uuid::Uuid;

type HmacSha256 = Hmac<Sha256>;

/// User represents a registered user stored in PostgreSQL with a Bcrypt password hash.
#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct User {
    pub id: Uuid,
    pub username: String,
    pub role: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// RegisterRequest represents the payload for registering a new user account (POST /oauth/register).
#[derive(Debug, Clone, Deserialize)]
pub struct RegisterRequest {
    pub username: String,
    pub password: String,
    #[serde(default)]
    pub role: String,
}

/// Standard RFC 7519 JWT payload claims.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub role: Option<String>,
    pub scope: String,
    pub iss: String,
    pub iat: i64,
    pub exp: i64,
}

/// OAuth 2.0 Token Request body (supports both JSON and application/x-www-form-urlencoded).
#[derive(Debug, Clone, Default, Deserialize)]
pub struct TokenRequest {
    #[serde(default)]
    pub grant_type: String,
    #[serde(default)]
    pub username: String,
    #[serde(default)]
    pub password: String,
    #[serde(default)]
    pub client_id: String,
    #[serde(default)]
    pub client_secret: String,
    #[serde(default)]
    pub scope: String,
}

/// Standard RFC 6749 OAuth 2.0 Access Token response.
#[derive(Debug, Clone, Serialize)]
pub struct TokenResponse {
    pub access_token: String,
    pub token_type: String,
    pub expires_in: u64,
    pub scope: String,
}

pub fn generate_jwt(
    secret: &str,
    subject: &str,
    role: Option<String>,
    scope: &str,
    ttl_secs: u64,
) -> anyhow::Result<String> {
    let header_b64 = URL_SAFE_NO_PAD.encode(r#"{"alg":"HS256","typ":"JWT"}"#.as_bytes());
    let now = Utc::now().timestamp();
    let claims = Claims {
        sub: subject.to_string(),
        role,
        scope: scope.to_string(),
        iss: "inventory-order-service".to_string(),
        iat: now,
        exp: now + (ttl_secs as i64),
    };
    let claims_json = serde_json::to_vec(&claims)?;
    let claims_b64 = URL_SAFE_NO_PAD.encode(&claims_json);

    let unsigned = format!("{}.{}", header_b64, claims_b64);
    let sig = sign_hs256(&unsigned, secret)?;
    Ok(format!("{}.{}", unsigned, sig))
}

pub fn verify_jwt(token_str: &str, secret: &str) -> anyhow::Result<Claims> {
    let parts: Vec<&str> = token_str.split('.').collect();
    if parts.len() != 3 {
        anyhow::bail!("invalid or expired JWT token");
    }

    let unsigned = format!("{}.{}", parts[0], parts[1]);
    let expected_sig = sign_hs256(&unsigned, secret)?;
    if parts[2] != expected_sig {
        anyhow::bail!("invalid or expired JWT token");
    }

    let claims_bytes = URL_SAFE_NO_PAD
        .decode(parts[1])
        .map_err(|_| anyhow::anyhow!("invalid or expired JWT token"))?;
    let claims: Claims = serde_json::from_slice(&claims_bytes)
        .map_err(|_| anyhow::anyhow!("invalid or expired JWT token"))?;

    if Utc::now().timestamp() > claims.exp {
        anyhow::bail!("invalid or expired JWT token");
    }

    Ok(claims)
}

fn sign_hs256(unsigned: &str, secret: &str) -> anyhow::Result<String> {
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes())
        .map_err(|_| anyhow::anyhow!("invalid HMAC secret"))?;
    mac.update(unsigned.as_bytes());
    let result = mac.finalize().into_bytes();
    Ok(URL_SAFE_NO_PAD.encode(result))
}

#[derive(Clone)]
pub struct AuthState {
    pub cfg: AuthConfig,
    pub db: PgPool,
}

#[derive(Clone)]
pub struct Handler {
    state: AuthState,
}

impl Handler {
    pub fn new(cfg: AuthConfig, db: PgPool) -> Self {
        Self {
            state: AuthState { cfg, db },
        }
    }

    pub fn routes(&self) -> Router {
        Router::new()
            .route("/oauth/config", get(get_auth_config))
            .route("/oauth/register", post(register_user))
            .route("/oauth/token", post(issue_token))
            .with_state(self.state.clone())
    }
}

async fn get_auth_config(State(state): State<AuthState>) -> Response {
    (
        StatusCode::OK,
        Json(json!({ "enable_register": state.cfg.enable_register })),
    )
        .into_response()
}

async fn register_user(
    State(state): State<AuthState>,
    payload: Result<Json<RegisterRequest>, JsonRejection>,
) -> Response {
    if !state.cfg.enable_register {
        return (
            StatusCode::FORBIDDEN,
            Json(json!({ "error": "user registration is disabled" })),
        )
            .into_response();
    }

    let Json(req) = match payload {
        Ok(r) => r,
        Err(_) => {
            return (
                StatusCode::BAD_REQUEST,
                Json(json!({ "error": "cannot parse JSON" })),
            )
                .into_response();
        }
    };

    let username = req.username.trim();
    if username.is_empty() || req.password.len() < 6 {
        return (
            StatusCode::BAD_REQUEST,
            Json(json!({ "error": "username and password (min 6 chars) are required" })),
        )
            .into_response();
    }

    let role = if req.role.trim().is_empty() {
        "user"
    } else {
        req.role.trim()
    };

    let res: Result<User, sqlx::Error> = sqlx::query_as(
        r#"
        INSERT INTO users (username, password_hash, role)
        VALUES ($1, crypt($2, gen_salt('bf', 10)), $3)
        RETURNING id, username, role, created_at, updated_at
        "#,
    )
    .bind(username)
    .bind(&req.password)
    .bind(role)
    .fetch_one(&state.db)
    .await;

    match res {
        Ok(user) => (StatusCode::CREATED, Json(user)).into_response(),
        Err(_) => (
            StatusCode::CONFLICT,
            Json(json!({ "error": "username already exists" })),
        )
            .into_response(),
    }
}

async fn issue_token(
    State(state): State<AuthState>,
    headers: HeaderMap,
    body: Bytes,
) -> Response {
    let content_type = headers
        .get(header::CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    let mut req: TokenRequest = if content_type.contains("application/x-www-form-urlencoded") {
        match serde_urlencoded::from_bytes(&body) {
            Ok(r) => r,
            Err(_) => {
                return (
                    StatusCode::BAD_REQUEST,
                    Json(json!({
                        "error": "invalid_request",
                        "error_description": "cannot parse OAuth 2.0 token request"
                    })),
                )
                    .into_response();
            }
        }
    } else {
        match serde_json::from_slice(&body) {
            Ok(r) => r,
            Err(_) => {
                if let Ok(r) = serde_urlencoded::from_bytes(&body) {
                    r
                } else {
                    return (
                        StatusCode::BAD_REQUEST,
                        Json(json!({
                            "error": "invalid_request",
                            "error_description": "cannot parse OAuth 2.0 token request"
                        })),
                    )
                        .into_response();
                }
            }
        }
    };

    // Support HTTP Basic Auth header for client_id / client_secret
    if let Some(auth_header) = headers
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
    {
        if let Some(b64) = auth_header.strip_prefix("Basic ") {
            if let Ok(decoded) = STANDARD.decode(b64) {
                if let Ok(decoded_str) = String::from_utf8(decoded) {
                    if let Some((cid, csec)) = decoded_str.split_once(':') {
                        req.client_id = cid.to_string();
                        req.client_secret = csec.to_string();
                    }
                }
            }
        }
    }

    let mut scope = if req.scope.is_empty() {
        "inventory:read inventory:write orders:read orders:write".to_string()
    } else {
        req.scope.clone()
    };

    let (subject, role) = match req.grant_type.as_str() {
        "password" => {
            let user_opt: Option<User> = sqlx::query_as(
                r#"
                SELECT id, username, role, created_at, updated_at
                FROM users
                WHERE username = $1 AND password_hash = crypt($2, password_hash)
                "#,
            )
            .bind(&req.username)
            .bind(&req.password)
            .fetch_optional(&state.db)
            .await
            .unwrap_or(None);

            let Some(user) = user_opt else {
                return (
                    StatusCode::UNAUTHORIZED,
                    Json(json!({
                        "error": "invalid_grant",
                        "error_description": "invalid username or password"
                    })),
                )
                    .into_response();
            };

            (user.username, Some(user.role))
        }
        "client_credentials" => {
            let row_opt: Option<(String, String)> = sqlx::query_as(
                r#"
                SELECT client_id, scope
                FROM oauth_clients
                WHERE client_id = $1 AND client_secret_hash = crypt($2, client_secret_hash)
                "#,
            )
            .bind(&req.client_id)
            .bind(&req.client_secret)
            .fetch_optional(&state.db)
            .await
            .unwrap_or(None);

            let Some((client_id, client_scope)) = row_opt else {
                return (
                    StatusCode::UNAUTHORIZED,
                    Json(json!({
                        "error": "invalid_client",
                        "error_description": "invalid client_id or client_secret"
                    })),
                )
                    .into_response();
            };

            if req.scope.is_empty() && !client_scope.is_empty() {
                scope = client_scope;
            }
            (client_id, Some("service".to_string()))
        }
        _ => {
            return (
                StatusCode::BAD_REQUEST,
                Json(json!({
                    "error": "unsupported_grant_type",
                    "error_description": "supported grant_types are 'password' and 'client_credentials'"
                })),
            )
                .into_response();
        }
    };

    let ttl = state.cfg.token_ttl_secs();
    match generate_jwt(&state.cfg.jwt_secret, &subject, role, &scope, ttl) {
        Ok(token) => (
            StatusCode::OK,
            Json(TokenResponse {
                access_token: token,
                token_type: "Bearer".to_string(),
                expires_in: ttl,
                scope,
            }),
        )
            .into_response(),
        Err(_) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({ "error": "server_error" })),
        )
            .into_response(),
    }
}

pub async fn jwt_middleware(
    State(cfg): State<AuthConfig>,
    req: Request,
    next: Next,
) -> Response {
    let auth_header = req
        .headers()
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok());

    let Some(token_str) = auth_header.and_then(|h| h.strip_prefix("Bearer ")) else {
        return (
            StatusCode::UNAUTHORIZED,
            Json(json!({ "error": "missing or invalid Authorization Bearer token" })),
        )
            .into_response();
    };

    if let Err(err) = verify_jwt(token_str, &cfg.jwt_secret) {
        return (
            StatusCode::UNAUTHORIZED,
            Json(json!({ "error": err.to_string() })),
        )
            .into_response();
    }

    next.run(req).await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_and_verify_jwt() {
        let secret = "super-secret-jwt-key-for-unit-test";

        // 1. Valid JWT generation and verification
        let token = generate_jwt(
            secret,
            "admin",
            Some("admin".to_string()),
            "inventory:read inventory:write",
            600,
        )
        .expect("failed to generate jwt");

        let claims = verify_jwt(&token, secret).expect("failed to verify valid jwt");
        assert_eq!(claims.sub, "admin");
        assert_eq!(claims.role.as_deref(), Some("admin"));

        // 2. Rejects wrong secret
        assert!(verify_jwt(&token, "wrong-secret").is_err());

        // 3. Rejects malformed token
        assert!(verify_jwt("invalid.token", secret).is_err());
    }
}

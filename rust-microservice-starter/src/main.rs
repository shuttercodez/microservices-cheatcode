use axum::http::StatusCode;
use axum::response::IntoResponse;
use axum::routing::get;
use axum::{Json, Router};
use rust_microservice_starter::config::Config;
use rust_microservice_starter::order;
use rust_microservice_starter::platform::cache::RedisCache;
use rust_microservice_starter::platform::database;
use rust_microservice_starter::platform::queue::RabbitMQ;
use rust_microservice_starter::product;
use serde_json::json;
use std::env;
use std::process;
use tokio::net::TcpListener;
use tokio::signal;
use tokio::sync::watch;
use tower_http::catch_panic::CatchPanicLayer;
use tower_http::cors::CorsLayer;
use tracing::{error, info};

#[tokio::main]
async fn main() {
    // Initialize structured JSON logger following microservice standards
    tracing_subscriber::fmt()
        .json()
        .with_current_span(false)
        .init();

    // 0. Load configuration from YAML (with Cloud Vault / Environment Variable overrides)
    let config_path = env::var("CONFIG_PATH")
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| "config/config.yaml".to_string());

    let cfg = match Config::load(&config_path) {
        Ok(c) => c,
        Err(err) => {
            error!(path = %config_path, error = %err, "Failed to load application configuration");
            process::exit(1);
        }
    };

    // 1. Connect to PostgreSQL using SQLx
    let db = match database::new_postgres_db(&cfg.database.dsn()).await {
        Ok(pool) => pool,
        Err(err) => {
            error!(error = %err, "Failed to connect to PostgreSQL");
            process::exit(1);
        }
    };

    // 2. Connect to Redis Cache
    let redis_cache = match RedisCache::new(&cfg.redis.addr).await {
        Ok(r) => r,
        Err(err) => {
            error!(error = %err, "Failed to connect to Redis");
            process::exit(1);
        }
    };

    // 3. Connect to RabbitMQ and declare queues
    let mq = match RabbitMQ::new(&cfg.rabbitmq.url).await {
        Ok(q) => q,
        Err(err) => {
            error!(error = %err, "Failed to connect to RabbitMQ");
            process::exit(1);
        }
    };

    if let Err(err) = mq.declare_queue(product::QUEUE_STOCK_CHANGED).await {
        error!(error = %err, "Failed to declare product queue");
        process::exit(1);
    }
    if let Err(err) = mq.declare_queue(order::QUEUE_ORDER_CREATED).await {
        error!(error = %err, "Failed to declare order queue");
        process::exit(1);
    }

    // 4. Start background queue consumers
    let (shutdown_tx, shutdown_rx) = watch::channel(false);
    tokio::spawn(product::start_queue_consumer(
        shutdown_rx.clone(),
        mq.conn.clone(),
        product::QUEUE_STOCK_CHANGED,
    ));
    tokio::spawn(order::start_queue_consumer(
        shutdown_rx,
        mq.conn.clone(),
        order::QUEUE_ORDER_CREATED,
    ));

    // 5. Dependency Injection (Repository, Cache & Producer -> Service -> Handler)
    // --- Product Domain ---
    let product_repo = product::new_repository(db.clone());
    let product_cache = product::new_cache(Some(redis_cache.clone()), cfg.redis.ttl());
    let product_producer = product::new_producer(Some(mq.clone()));
    let product_service = product::new_service(product_repo, Some(product_cache), product_producer);
    let product_handler = product::Handler::new(product_service.clone());

    // --- Order Domain (Inject product_service for cross-domain stock verification and deduction) ---
    let order_repo = order::new_repository(db.clone());
    let order_cache = order::new_cache(Some(redis_cache.clone()), cfg.redis.ttl());
    let order_producer = order::new_producer(Some(mq.clone()));
    let order_service =
        order::new_service(order_repo, Some(order_cache), product_service, order_producer);
    let order_handler = order::Handler::new(order_service);

    // 6. Configure Axum application
    let db_ready = db.clone();
    let redis_ready = redis_cache.clone();
    let mq_ready = mq.clone();

    let app = Router::new()
        .route(
            "/healthz",
            get(|| async { (StatusCode::OK, Json(json!({ "status": "alive" }))) }),
        )
        .route(
            "/readyz",
            get(move || {
                let db = db_ready.clone();
                let redis = redis_ready.clone();
                let mq = mq_ready.clone();
                async move {
                    if database::ping(&db).await.is_err()
                        || redis.ping().await.is_err()
                        || mq.is_closed()
                    {
                        (
                            StatusCode::SERVICE_UNAVAILABLE,
                            Json(json!({ "status": "not ready" })),
                        )
                            .into_response()
                    } else {
                        (StatusCode::OK, Json(json!({ "status": "ready" }))).into_response()
                    }
                }
            }),
        )
        .merge(product_handler.routes())
        .merge(order_handler.routes())
        .layer(CorsLayer::permissive())
        .layer(CatchPanicLayer::new());

    // 7. Start Axum server and configure graceful shutdown
    let listen_addr = format!("0.0.0.0:{}", cfg.app.port);
    let listener = match TcpListener::bind(&listen_addr).await {
        Ok(l) => l,
        Err(err) => {
            error!(error = %err, "Failed to bind TCP listener");
            process::exit(1);
        }
    };

    info!(
        app = %cfg.app.name,
        env = %cfg.app.env,
        addr = %format!(":{}", cfg.app.port),
        "Starting Axum HTTP server"
    );

    let shutdown_timeout = cfg.app.shutdown_timeout();
    let server = axum::serve(listener, app).with_graceful_shutdown(async move {
        shutdown_signal().await;
        info!("Shutting down service gracefully...");
        let _ = shutdown_tx.send(true);
    });

    if let Err(err) = tokio::time::timeout(shutdown_timeout, server).await {
        error!(error = %err, "Axum forced to shutdown");
    }

    mq.close().await;
    info!("Service exited cleanly");
}

async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
}

use crate::order::model::{CreateOrderRequest, OrderError};
use crate::order::service::Service;
use crate::product::ProductError;
use axum::extract::rejection::JsonRejection;
use axum::extract::{Path, Query, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::routing::get;
use axum::{Json, Router};
use serde::Deserialize;
use serde_json::json;
use std::sync::Arc;

#[derive(Debug, Deserialize)]
pub struct PaginationQuery {
    #[serde(default = "default_page")]
    pub page: i64,
    #[serde(default = "default_limit")]
    pub limit: i64,
}

fn default_page() -> i64 {
    1
}
fn default_limit() -> i64 {
    10
}

/// Handler manages HTTP requests and responses for order endpoints.
#[derive(Clone)]
pub struct Handler {
    service: Arc<dyn Service>,
}

impl Handler {
    pub fn new(service: Arc<dyn Service>) -> Self {
        Self { service }
    }

    pub fn routes(&self) -> Router {
        Router::new()
            .route("/api/v1/orders", get(get_all).post(create))
            .route("/api/v1/orders/", get(get_all).post(create))
            .route("/api/v1/orders/{id}", get(get_by_id))
            .with_state(self.service.clone())
    }
}

async fn create(
    State(service): State<Arc<dyn Service>>,
    payload: Result<Json<CreateOrderRequest>, JsonRejection>,
) -> Response {
    let Json(req) = match payload {
        Ok(req) => req,
        Err(_) => {
            return (
                StatusCode::BAD_REQUEST,
                Json(json!({ "error": "cannot parse JSON" })),
            )
                .into_response();
        }
    };

    match service.create_order(req).await {
        Ok(order) => (StatusCode::CREATED, Json(order)).into_response(),
        Err(err) => {
            let status = match err {
                OrderError::InvalidInput => StatusCode::BAD_REQUEST,
                OrderError::Product(ProductError::NotFound) => StatusCode::NOT_FOUND,
                OrderError::InsufficientStock
                | OrderError::Product(ProductError::VersionConflict) => StatusCode::CONFLICT,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

async fn get_all(
    State(service): State<Arc<dyn Service>>,
    Query(query): Query<PaginationQuery>,
) -> Response {
    match service.get_all_orders(query.page, query.limit).await {
        Ok(orders) => (StatusCode::OK, Json(orders)).into_response(),
        Err(err) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({ "error": err.to_string() })),
        )
            .into_response(),
    }
}

async fn get_by_id(
    State(service): State<Arc<dyn Service>>,
    Path(id): Path<String>,
) -> Response {
    match service.get_order_by_id(&id).await {
        Ok(order) => (StatusCode::OK, Json(order)).into_response(),
        Err(err) => {
            let status = match err {
                OrderError::NotFound => StatusCode::NOT_FOUND,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

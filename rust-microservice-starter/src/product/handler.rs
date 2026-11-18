use crate::product::model::{CreateProductRequest, PatchStockRequest, ProductError};
use crate::product::service::Service;
use axum::extract::rejection::JsonRejection;
use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::routing::{get, patch};
use axum::{Json, Router};
use serde_json::json;
use std::sync::Arc;

/// Handler manages HTTP requests and responses for product endpoints.
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
            .route("/api/v1/products", get(get_all).post(create))
            .route("/api/v1/products/", get(get_all).post(create))
            .route("/api/v1/products/{id}", get(get_by_id).put(update).delete(delete))
            .route("/api/v1/products/{id}/stock", patch(patch_stock))
            .with_state(self.service.clone())
    }
}

async fn create(
    State(service): State<Arc<dyn Service>>,
    payload: Result<Json<CreateProductRequest>, JsonRejection>,
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

    match service.create(req).await {
        Ok(product) => (StatusCode::CREATED, Json(product)).into_response(),
        Err(err) => {
            let status = match err {
                ProductError::InvalidPayload => StatusCode::BAD_REQUEST,
                _ => StatusCode::CONFLICT,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

async fn get_all(State(service): State<Arc<dyn Service>>) -> Response {
    match service.get_all().await {
        Ok(products) => (StatusCode::OK, Json(products)).into_response(),
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
    match service.get_by_id(&id).await {
        Ok(product) => (StatusCode::OK, Json(product)).into_response(),
        Err(err) => {
            let status = match err {
                ProductError::NotFound => StatusCode::NOT_FOUND,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

async fn update(
    State(service): State<Arc<dyn Service>>,
    Path(id): Path<String>,
    payload: Result<Json<CreateProductRequest>, JsonRejection>,
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

    match service.update(&id, req).await {
        Ok(product) => (StatusCode::OK, Json(product)).into_response(),
        Err(err) => {
            let status = match err {
                ProductError::NotFound => StatusCode::NOT_FOUND,
                ProductError::InvalidPayload => StatusCode::BAD_REQUEST,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

async fn patch_stock(
    State(service): State<Arc<dyn Service>>,
    Path(id): Path<String>,
    payload: Result<Json<PatchStockRequest>, JsonRejection>,
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

    match service.patch_stock(&id, req).await {
        Ok(product) => (StatusCode::OK, Json(product)).into_response(),
        Err(err) => {
            let status = match err {
                ProductError::NegativeStock => StatusCode::BAD_REQUEST,
                ProductError::NotFound => StatusCode::NOT_FOUND,
                ProductError::VersionConflict => StatusCode::CONFLICT,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

async fn delete(State(service): State<Arc<dyn Service>>, Path(id): Path<String>) -> Response {
    match service.delete(&id).await {
        Ok(()) => StatusCode::NO_CONTENT.into_response(),
        Err(err) => {
            let status = match err {
                ProductError::NotFound => StatusCode::NOT_FOUND,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            (status, Json(json!({ "error": err.to_string() }))).into_response()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::product::model::Product;
    use crate::product::service::new_service;
    use crate::product::service::tests::{MockCache, MockProducer, MockRepository};
    use axum::body::Body;
    use axum::http::{Method, Request};
    use http_body_util::BodyExt;
    use tower::ServiceExt;

    #[tokio::test]
    async fn test_product_handler_endpoints() {
        let repo = MockRepository::new();
        let cache = MockCache::new();
        let producer = MockProducer::new();
        let svc = new_service(repo, Some(cache), producer);
        let handler = Handler::new(svc);
        let app = handler.routes();

        // 1. POST /api/v1/products/ - creates product via HTTP (201 Created)
        let payload = r#"{"sku":"IPHONE-16","name":"iPhone 16 Pro","price":39900,"stock":50}"#;
        let req = Request::builder()
            .method(Method::POST)
            .uri("/api/v1/products/")
            .header("Content-Type", "application/json")
            .body(Body::from(payload))
            .unwrap();

        let resp = app.clone().oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::CREATED);

        let body_bytes = resp.into_body().collect().await.unwrap().to_bytes();
        let created_product: Product = serde_json::from_slice(&body_bytes).unwrap();
        let created_id = created_product.id.to_string();
        assert_eq!(created_product.sku, "IPHONE-16");
        assert_eq!(created_product.stock, 50);

        // 2. GET /api/v1/products/:id - retrieves newly created product (200 OK)
        let req = Request::builder()
            .method(Method::GET)
            .uri(format!("/api/v1/products/{}", created_id))
            .body(Body::empty())
            .unwrap();

        let resp = app.clone().oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::OK);

        // 3. PATCH /api/v1/products/:id/stock - updates product stock (200 OK)
        let patch_payload = r#"{"stock":45}"#;
        let req = Request::builder()
            .method(Method::PATCH)
            .uri(format!("/api/v1/products/{}/stock", created_id))
            .header("Content-Type", "application/json")
            .body(Body::from(patch_payload))
            .unwrap();

        let resp = app.oneshot(req).await.unwrap();
        assert_eq!(resp.status(), StatusCode::OK);

        let body_bytes = resp.into_body().collect().await.unwrap().to_bytes();
        let updated_product: Product = serde_json::from_slice(&body_bytes).unwrap();
        assert_eq!(updated_product.stock, 45);
        assert_eq!(updated_product.version, 2);
    }
}

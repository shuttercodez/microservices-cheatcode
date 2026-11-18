use crate::order::model::{CreateOrderRequest, Order, OrderError};
use crate::order::producer::Producer;
use crate::order::repository::Repository;
use crate::product::{self, PatchStockRequest};
use async_trait::async_trait;
use chrono::Utc;
use std::sync::Arc;
use uuid::Uuid;

/// Service defines the business logic contract for managing orders.
#[async_trait]
pub trait Service: Send + Sync {
    async fn create_order(&self, req: CreateOrderRequest) -> Result<Order, OrderError>;
    async fn get_all_orders(&self) -> Result<Vec<Order>, OrderError>;
    async fn get_order_by_id(&self, id: &str) -> Result<Order, OrderError>;
}

pub struct OrderService {
    order_repo: Arc<dyn Repository>,
    product_service: Arc<dyn product::Service>,
    producer: Arc<dyn Producer>,
}

pub fn new_service(
    order_repo: Arc<dyn Repository>,
    product_service: Arc<dyn product::Service>,
    producer: Arc<dyn Producer>,
) -> Arc<dyn Service> {
    Arc::new(OrderService {
        order_repo,
        product_service,
        producer,
    })
}

#[async_trait]
impl Service for OrderService {
    /// Verifies product availability, deducts stock via ProductService, persists the order, and publishes an event.
    async fn create_order(&self, req: CreateOrderRequest) -> Result<Order, OrderError> {
        if req.product_id.is_empty() || req.quantity <= 0 {
            return Err(OrderError::InvalidInput);
        }

        // 1. Fetch product details and verify stock availability via ProductService
        let prod = self.product_service.get_by_id(&req.product_id).await?;

        if prod.stock < req.quantity {
            return Err(OrderError::InsufficientStock);
        }

        // 2. Deduct stock via ProductService.patch_stock (enforcing optimistic locking)
        let new_stock = prod.stock - req.quantity;
        self.product_service
            .patch_stock(&req.product_id, PatchStockRequest { stock: new_stock })
            .await?;

        // 3. Calculate total price and persist the order record
        let now = Utc::now();
        let mut order = Order {
            id: Uuid::nil(),
            product_id: prod.id,
            quantity: req.quantity,
            total_price: prod.price * (req.quantity as f64),
            status: "COMPLETED".to_string(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.order_repo.create(&mut order).await?;

        // 4. Publish order created event to RabbitMQ
        let _ = self.producer.publish_order_created(&order).await;

        Ok(order)
    }

    async fn get_all_orders(&self) -> Result<Vec<Order>, OrderError> {
        self.order_repo.find_all().await
    }

    async fn get_order_by_id(&self, id: &str) -> Result<Order, OrderError> {
        self.order_repo.find_by_id(id).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::product::{CreateProductRequest, Product, ProductError};
    use std::collections::HashMap;
    use std::sync::atomic::{AtomicBool, Ordering};
    use tokio::sync::RwLock;

    struct MockOrderRepository {
        orders: RwLock<HashMap<String, Order>>,
    }

    impl MockOrderRepository {
        fn new() -> Arc<Self> {
            Arc::new(Self {
                orders: RwLock::new(HashMap::new()),
            })
        }
    }

    #[async_trait]
    impl Repository for MockOrderRepository {
        async fn create(&self, order: &mut Order) -> Result<(), OrderError> {
            if order.id.is_nil() {
                order.id = Uuid::new_v4();
            }
            self.orders
                .write()
                .await
                .insert(order.id.to_string(), order.clone());
            Ok(())
        }

        async fn find_all(&self) -> Result<Vec<Order>, OrderError> {
            Ok(self.orders.read().await.values().cloned().collect())
        }

        async fn find_by_id(&self, id: &str) -> Result<Order, OrderError> {
            self.orders
                .read()
                .await
                .get(id)
                .cloned()
                .ok_or(OrderError::NotFound)
        }
    }

    struct MockProductService {
        products: RwLock<HashMap<String, Product>>,
        patch_stock_called: AtomicBool,
        force_patch_err: RwLock<Option<ProductError>>,
    }

    impl MockProductService {
        fn new() -> Arc<Self> {
            Arc::new(Self {
                products: RwLock::new(HashMap::new()),
                patch_stock_called: AtomicBool::new(false),
                force_patch_err: RwLock::new(None),
            })
        }
    }

    #[async_trait]
    impl product::Service for MockProductService {
        async fn create(&self, _req: CreateProductRequest) -> Result<Product, ProductError> {
            Err(ProductError::NotFound)
        }

        async fn get_all(&self) -> Result<Vec<Product>, ProductError> {
            Ok(vec![])
        }

        async fn get_by_id(&self, id: &str) -> Result<Product, ProductError> {
            self.products
                .read()
                .await
                .get(id)
                .cloned()
                .ok_or(ProductError::NotFound)
        }

        async fn update(
            &self,
            _id: &str,
            _req: CreateProductRequest,
        ) -> Result<Product, ProductError> {
            Err(ProductError::NotFound)
        }

        async fn patch_stock(
            &self,
            id: &str,
            req: PatchStockRequest,
        ) -> Result<Product, ProductError> {
            self.patch_stock_called.store(true, Ordering::SeqCst);
            if let Some(ref err) = *self.force_patch_err.read().await {
                return Err(err.clone());
            }
            let mut map = self.products.write().await;
            let p = map.get_mut(id).ok_or(ProductError::NotFound)?;
            p.stock = req.stock;
            p.version += 1;
            Ok(p.clone())
        }

        async fn delete(&self, _id: &str) -> Result<(), ProductError> {
            Ok(())
        }
    }

    struct MockOrderProducer {
        published_orders: RwLock<Vec<Order>>,
    }

    impl MockOrderProducer {
        fn new() -> Arc<Self> {
            Arc::new(Self {
                published_orders: RwLock::new(Vec::new()),
            })
        }
    }

    #[async_trait]
    impl Producer for MockOrderProducer {
        async fn publish_order_created(&self, order: &Order) -> anyhow::Result<()> {
            self.published_orders.write().await.push(order.clone());
            Ok(())
        }
    }

    #[tokio::test]
    async fn test_order_service_create_order() {
        let product_id = Uuid::new_v4();
        let now = Utc::now();

        // 1. Success - creates order, deducts product stock, calculates total price, and publishes event
        {
            let order_repo = MockOrderRepository::new();
            let prod_svc = MockProductService::new();
            let producer = MockOrderProducer::new();

            prod_svc.products.write().await.insert(
                product_id.to_string(),
                Product {
                    id: product_id,
                    sku: "KEY-001".to_string(),
                    name: "Mechanical Keyboard".to_string(),
                    price: 1500.0,
                    stock: 10,
                    version: 1,
                    created_at: now,
                    updated_at: now,
                    deleted_at: None,
                },
            );

            let svc = new_service(order_repo, prod_svc.clone(), producer.clone());

            let order = svc
                .create_order(CreateOrderRequest {
                    product_id: product_id.to_string(),
                    quantity: 3,
                })
                .await
                .expect("expected order creation success");

            assert_eq!(order.total_price, 4500.0);
            assert_eq!(
                prod_svc
                    .products
                    .read()
                    .await
                    .get(&product_id.to_string())
                    .unwrap()
                    .stock,
                7
            );
            assert_eq!(producer.published_orders.read().await.len(), 1);
        }

        // 2. Fail - ordering more than available stock returns InsufficientStock without deducting stock
        {
            let order_repo = MockOrderRepository::new();
            let prod_svc = MockProductService::new();
            let producer = MockOrderProducer::new();

            prod_svc.products.write().await.insert(
                product_id.to_string(),
                Product {
                    id: product_id,
                    sku: "KEY-001".to_string(),
                    name: "Mechanical Keyboard".to_string(),
                    price: 1500.0,
                    stock: 2,
                    version: 1,
                    created_at: now,
                    updated_at: now,
                    deleted_at: None,
                },
            );

            let svc = new_service(order_repo, prod_svc.clone(), producer.clone());

            let err = svc
                .create_order(CreateOrderRequest {
                    product_id: product_id.to_string(),
                    quantity: 5,
                })
                .await
                .unwrap_err();

            assert_eq!(err, OrderError::InsufficientStock);
            assert!(!prod_svc.patch_stock_called.load(Ordering::SeqCst));
            assert!(producer.published_orders.read().await.is_empty());
        }

        // 3. Fail - non-existent product returns ProductError::NotFound
        {
            let order_repo = MockOrderRepository::new();
            let prod_svc = MockProductService::new();
            let producer = MockOrderProducer::new();
            let svc = new_service(order_repo, prod_svc, producer);

            let err = svc
                .create_order(CreateOrderRequest {
                    product_id: Uuid::new_v4().to_string(),
                    quantity: 1,
                })
                .await
                .unwrap_err();

            assert_eq!(err, OrderError::Product(ProductError::NotFound));
        }
    }
}

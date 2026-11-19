use crate::product::cache::Cache;
use crate::product::model::{
    CreateProductRequest, PaginatedProducts, PatchStockRequest, Product, ProductError,
};
use crate::product::producer::Producer;
use crate::product::repository::Repository;
use async_trait::async_trait;
use chrono::Utc;
use std::sync::Arc;
use uuid::Uuid;

/// Service defines the business logic contract for managing products.
#[async_trait]
pub trait Service: Send + Sync {
    async fn create(&self, req: CreateProductRequest) -> Result<Product, ProductError>;
    async fn get_all(&self, page: i64, limit: i64) -> Result<PaginatedProducts, ProductError>;
    async fn get_by_id(&self, id: &str) -> Result<Product, ProductError>;
    async fn update(&self, id: &str, req: CreateProductRequest) -> Result<Product, ProductError>;
    async fn patch_stock(&self, id: &str, req: PatchStockRequest) -> Result<Product, ProductError>;
    async fn delete(&self, id: &str) -> Result<(), ProductError>;
}

pub struct ProductService {
    repo: Arc<dyn Repository>,
    cache: Option<Arc<dyn Cache>>,
    producer: Arc<dyn Producer>,
}

pub fn new_service(
    repo: Arc<dyn Repository>,
    cache: Option<Arc<dyn Cache>>,
    producer: Arc<dyn Producer>,
) -> Arc<dyn Service> {
    Arc::new(ProductService {
        repo,
        cache,
        producer,
    })
}

fn normalize_pagination(page: i64, limit: i64) -> (i64, i64) {
    let page = if page < 1 { 1 } else { page };
    let limit = if limit < 1 {
        10
    } else if limit > 100 {
        100
    } else {
        limit
    };
    (page, limit)
}

fn calc_total_pages(total: i64, limit: i64) -> i64 {
    if total == 0 || limit <= 0 {
        0
    } else {
        (total + limit - 1) / limit
    }
}

#[async_trait]
impl Service for ProductService {
    async fn create(&self, req: CreateProductRequest) -> Result<Product, ProductError> {
        if req.sku.is_empty() || req.name.is_empty() || req.price < 0.0 || req.stock < 0 {
            return Err(ProductError::InvalidPayload);
        }

        let now = Utc::now();
        let mut product = Product {
            id: Uuid::nil(),
            sku: req.sku,
            name: req.name,
            price: req.price,
            stock: req.stock,
            version: 1,
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.repo.create(&mut product).await?;

        if let Some(ref cache) = self.cache {
            let _ = cache.set(&product).await;
            let _ = cache.invalidate_lists().await;
        }
        let _ = self.producer.publish_stock_changed(&product).await;

        Ok(product)
    }

    async fn get_all(&self, page: i64, limit: i64) -> Result<PaginatedProducts, ProductError> {
        let (page, limit) = normalize_pagination(page, limit);

        if let Some(ref cache) = self.cache {
            if let Ok(Some(cached)) = cache.get_list(page, limit).await {
                return Ok(cached);
            }
        }

        let (products, total) = self.repo.find_all(page, limit).await?;
        let res = PaginatedProducts {
            data: products,
            page,
            limit,
            total,
            total_pages: calc_total_pages(total, limit),
        };

        if let Some(ref cache) = self.cache {
            let _ = cache.set_list(page, limit, &res).await;
        }
        Ok(res)
    }

    /// Implements the Cache-Aside pattern: check Redis first, fallback to DB on miss, and populate Redis.
    async fn get_by_id(&self, id: &str) -> Result<Product, ProductError> {
        if let Some(ref cache) = self.cache {
            if let Ok(Some(cached)) = cache.get(id).await {
                return Ok(cached);
            }
        }

        let product = self.repo.find_by_id(id).await?;

        if let Some(ref cache) = self.cache {
            let _ = cache.set(&product).await;
        }
        Ok(product)
    }

    async fn update(&self, id: &str, req: CreateProductRequest) -> Result<Product, ProductError> {
        let mut product = self.repo.find_by_id(id).await?;

        if req.sku.is_empty() || req.name.is_empty() || req.price < 0.0 || req.stock < 0 {
            return Err(ProductError::InvalidPayload);
        }

        product.sku = req.sku;
        product.name = req.name;
        product.price = req.price;
        product.stock = req.stock;
        product.version += 1;

        self.repo.update(&mut product).await?;

        if let Some(ref cache) = self.cache {
            let _ = cache.set(&product).await;
            let _ = cache.invalidate_lists().await;
        }
        let _ = self.producer.publish_stock_changed(&product).await;

        Ok(product)
    }

    async fn patch_stock(
        &self,
        id: &str,
        req: PatchStockRequest,
    ) -> Result<Product, ProductError> {
        if req.stock < 0 {
            return Err(ProductError::NegativeStock);
        }

        // Always read current version directly from the database for optimistic locking
        let mut product = self.repo.find_by_id(id).await?;

        self.repo
            .update_stock_with_version(&mut product, req.stock)
            .await?;

        // Refresh Redis cache with the new stock and incremented version
        if let Some(ref cache) = self.cache {
            let _ = cache.set(&product).await;
            let _ = cache.invalidate_lists().await;
        }
        let _ = self.producer.publish_stock_changed(&product).await;

        Ok(product)
    }

    async fn delete(&self, id: &str) -> Result<(), ProductError> {
        self.repo.delete(id).await?;

        if let Some(ref cache) = self.cache {
            let _ = cache.delete(id).await;
            let _ = cache.invalidate_lists().await;
        }
        Ok(())
    }
}

#[cfg(test)]
pub mod tests {
    use super::*;
    use crate::product::cache::list_cache_key;
    use std::collections::HashMap;
    use std::sync::atomic::{AtomicUsize, Ordering};
    use tokio::sync::RwLock;

    pub struct MockRepository {
        pub products: RwLock<HashMap<String, Product>>,
        pub find_by_id_calls: AtomicUsize,
        pub force_create_err: RwLock<Option<ProductError>>,
        pub force_conflict_err: AtomicUsize,
    }

    impl MockRepository {
        pub fn new() -> Arc<Self> {
            Arc::new(Self {
                products: RwLock::new(HashMap::new()),
                find_by_id_calls: AtomicUsize::new(0),
                force_create_err: RwLock::new(None),
                force_conflict_err: AtomicUsize::new(0),
            })
        }
    }

    #[async_trait]
    impl Repository for MockRepository {
        async fn create(&self, product: &mut Product) -> Result<(), ProductError> {
            if let Some(ref err) = *self.force_create_err.read().await {
                return Err(err.clone());
            }
            if product.id.is_nil() {
                product.id = Uuid::new_v4();
            }
            product.version = 1;
            self.products
                .write()
                .await
                .insert(product.id.to_string(), product.clone());
            Ok(())
        }

        async fn find_all(
            &self,
            _page: i64,
            _limit: i64,
        ) -> Result<(Vec<Product>, i64), ProductError> {
            let map = self.products.read().await;
            let list: Vec<Product> = map.values().cloned().collect();
            let total = list.len() as i64;
            Ok((list, total))
        }

        async fn find_by_id(&self, id: &str) -> Result<Product, ProductError> {
            self.find_by_id_calls.fetch_add(1, Ordering::SeqCst);
            let map = self.products.read().await;
            map.get(id).cloned().ok_or(ProductError::NotFound)
        }

        async fn update(&self, product: &mut Product) -> Result<(), ProductError> {
            self.products
                .write()
                .await
                .insert(product.id.to_string(), product.clone());
            Ok(())
        }

        async fn update_stock_with_version(
            &self,
            product: &mut Product,
            new_stock: i32,
        ) -> Result<(), ProductError> {
            if self.force_conflict_err.load(Ordering::SeqCst) > 0 {
                return Err(ProductError::VersionConflict);
            }
            let mut map = self.products.write().await;
            let stored = map
                .get_mut(&product.id.to_string())
                .ok_or(ProductError::NotFound)?;
            if stored.version != product.version {
                return Err(ProductError::VersionConflict);
            }
            stored.stock = new_stock;
            stored.version += 1;

            product.stock = new_stock;
            product.version = stored.version;
            Ok(())
        }

        async fn delete(&self, id: &str) -> Result<(), ProductError> {
            let mut map = self.products.write().await;
            if map.remove(id).is_none() {
                return Err(ProductError::NotFound);
            }
            Ok(())
        }
    }

    pub struct MockCache {
        pub store: RwLock<HashMap<String, Product>>,
        pub list_store: RwLock<HashMap<String, PaginatedProducts>>,
    }

    impl MockCache {
        pub fn new() -> Arc<Self> {
            Arc::new(Self {
                store: RwLock::new(HashMap::new()),
                list_store: RwLock::new(HashMap::new()),
            })
        }
    }

    #[async_trait]
    impl Cache for MockCache {
        async fn get(&self, id: &str) -> anyhow::Result<Option<Product>> {
            Ok(self.store.read().await.get(id).cloned())
        }

        async fn set(&self, product: &Product) -> anyhow::Result<()> {
            self.store
                .write()
                .await
                .insert(product.id.to_string(), product.clone());
            Ok(())
        }

        async fn delete(&self, id: &str) -> anyhow::Result<()> {
            self.store.write().await.remove(id);
            Ok(())
        }

        async fn get_list(&self, page: i64, limit: i64) -> anyhow::Result<Option<PaginatedProducts>> {
            let key = list_cache_key(page, limit);
            Ok(self.list_store.read().await.get(&key).cloned())
        }

        async fn set_list(
            &self,
            page: i64,
            limit: i64,
            list: &PaginatedProducts,
        ) -> anyhow::Result<()> {
            let key = list_cache_key(page, limit);
            self.list_store.write().await.insert(key, list.clone());
            Ok(())
        }

        async fn invalidate_lists(&self) -> anyhow::Result<()> {
            self.list_store.write().await.clear();
            Ok(())
        }
    }

    pub struct MockProducer {
        pub published_events: RwLock<Vec<Product>>,
    }

    impl MockProducer {
        pub fn new() -> Arc<Self> {
            Arc::new(Self {
                published_events: RwLock::new(Vec::new()),
            })
        }
    }

    #[async_trait]
    impl Producer for MockProducer {
        async fn publish_stock_changed(&self, product: &Product) -> anyhow::Result<()> {
            self.published_events.write().await.push(product.clone());
            Ok(())
        }
    }

    #[tokio::test]
    async fn test_product_service_create() {
        // 1. Success - creates product, caches it, and publishes event to queue
        {
            let repo = MockRepository::new();
            let cache = MockCache::new();
            let producer = MockProducer::new();
            let svc = new_service(repo.clone(), Some(cache.clone()), producer.clone());

            let req = CreateProductRequest {
                sku: "LAPTOP-001".to_string(),
                name: "Gaming Laptop".to_string(),
                price: 35000.0,
                stock: 10,
            };

            let res = svc.create(req).await.expect("expected success");
            assert_eq!(res.sku, "LAPTOP-001");
            assert_eq!(producer.published_events.read().await.len(), 1);
        }

        // 2. Fail - negative price returns InvalidPayload and does not publish event
        {
            let repo = MockRepository::new();
            let cache = MockCache::new();
            let producer = MockProducer::new();
            let svc = new_service(repo, Some(cache), producer.clone());

            let req = CreateProductRequest {
                sku: "LAPTOP-002".to_string(),
                name: "Broken Laptop".to_string(),
                price: -100.0,
                stock: 5,
            };

            let err = svc.create(req).await.unwrap_err();
            assert_eq!(err, ProductError::InvalidPayload);
            assert!(producer.published_events.read().await.is_empty());
        }

        // 3. Fail - duplicate SKU returns SkuConflict
        {
            let repo = MockRepository::new();
            *repo.force_create_err.write().await = Some(ProductError::SkuConflict);
            let cache = MockCache::new();
            let producer = MockProducer::new();
            let svc = new_service(repo, Some(cache), producer.clone());

            let req = CreateProductRequest {
                sku: "DUP-SKU".to_string(),
                name: "Duplicate Item".to_string(),
                price: 500.0,
                stock: 1,
            };

            let err = svc.create(req).await.unwrap_err();
            assert_eq!(err, ProductError::SkuConflict);
            assert!(producer.published_events.read().await.is_empty());
        }
    }

    #[tokio::test]
    async fn test_product_service_cache_aside_get_by_id() {
        let repo = MockRepository::new();
        let cache = MockCache::new();
        let producer = MockProducer::new();
        let svc = new_service(repo.clone(), Some(cache.clone()), producer);

        let id = Uuid::new_v4();
        let now = Utc::now();
        repo.products.write().await.insert(
            id.to_string(),
            Product {
                id,
                sku: "CACHE-001".to_string(),
                name: "Cached Item".to_string(),
                price: 1200.0,
                stock: 5,
                version: 1,
                created_at: now,
                updated_at: now,
                deleted_at: None,
            },
        );

        // 1st Call: Cache Miss -> should query DB (find_by_id_calls == 1) and populate Redis cache
        let p1 = svc
            .get_by_id(&id.to_string())
            .await
            .expect("1st GetByID failed");
        assert_eq!(p1.sku, "CACHE-001");
        assert_eq!(repo.find_by_id_calls.load(Ordering::SeqCst), 1);

        // 2nd Call: Cache Hit -> should return from Redis WITHOUT querying DB again (find_by_id_calls stays 1)
        let p2 = svc
            .get_by_id(&id.to_string())
            .await
            .expect("2nd GetByID failed");
        assert_eq!(p2.sku, "CACHE-001");
        assert_eq!(repo.find_by_id_calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn test_product_service_patch_stock_optimistic_locking() {
        let repo = MockRepository::new();
        let cache = MockCache::new();
        let producer = MockProducer::new();
        let svc = new_service(repo.clone(), Some(cache.clone()), producer.clone());

        let created = svc
            .create(CreateProductRequest {
                sku: "MOUSE-01".to_string(),
                name: "Wireless Mouse".to_string(),
                price: 990.0,
                stock: 20,
            })
            .await
            .unwrap();

        producer.published_events.write().await.clear();

        // 1. Success - updates stock, refreshes cache, and increments version from 1 to 2
        let updated = svc
            .patch_stock(&created.id.to_string(), PatchStockRequest { stock: 15 })
            .await
            .expect("patch_stock failed");
        assert_eq!(updated.stock, 15);
        assert_eq!(updated.version, 2);

        let cached = cache.get(&created.id.to_string()).await.unwrap().unwrap();
        assert_eq!(cached.stock, 15);
        assert_eq!(producer.published_events.read().await.len(), 1);

        // 2. Fail - negative stock returns NegativeStock
        let err = svc
            .patch_stock(&created.id.to_string(), PatchStockRequest { stock: -5 })
            .await
            .unwrap_err();
        assert_eq!(err, ProductError::NegativeStock);

        // 3. Fail - concurrent modification returns VersionConflict
        repo.force_conflict_err.store(1, Ordering::SeqCst);
        let err_conflict = svc
            .patch_stock(&created.id.to_string(), PatchStockRequest { stock: 10 })
            .await
            .unwrap_err();
        assert_eq!(err_conflict, ProductError::VersionConflict);
    }
}

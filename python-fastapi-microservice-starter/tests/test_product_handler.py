from __future__ import annotations

import pytest
from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient

from app.product import Handler, ProductService
from tests.test_product_service import MockCache, MockProducer, MockRepository


@pytest.mark.asyncio
async def test_product_handler_endpoints() -> None:
    repo = MockRepository()
    cache = MockCache()
    producer = MockProducer()
    svc = ProductService(repo, cache, producer)
    handler = Handler(svc)

    app = FastAPI()
    app.include_router(handler.router)

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        # 1. POST /api/v1/products/ - creates product via HTTP (201 Created)
        resp = await client.post(
            "/api/v1/products/",
            json={"sku": "IPHONE-16", "name": "iPhone 16 Pro", "price": 39900, "stock": 50},
        )
        assert resp.status_code == 201
        body = resp.json()
        created_id = body["id"]
        assert body["sku"] == "IPHONE-16"
        assert body["stock"] == 50

        # 2. GET /api/v1/products/:id - retrieves newly created product (200 OK)
        resp = await client.get(f"/api/v1/products/{created_id}")
        assert resp.status_code == 200

        # 3. PATCH /api/v1/products/:id/stock - updates product stock (200 OK)
        resp = await client.patch(
            f"/api/v1/products/{created_id}/stock",
            json={"stock": 45},
        )
        assert resp.status_code == 200
        body = resp.json()
        assert body["stock"] == 45
        assert body["version"] == 2

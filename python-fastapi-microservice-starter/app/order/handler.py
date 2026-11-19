from __future__ import annotations

from fastapi import APIRouter, Query, status
from fastapi.responses import JSONResponse

from app.order.model import (
    CreateOrderRequest,
    ErrInsufficientStock,
    ErrInvalidOrderInput,
    ErrOrderNotFound,
)
from app.order.service import Service
from app.product.model import ErrProductNotFound, ErrVersionConflict


class Handler:
    """Manages HTTP requests and responses for order endpoints."""

    def __init__(self, service: Service) -> None:
        self.service = service
        self.router = APIRouter(prefix="/api/v1/orders", tags=["orders"])
        self._register_routes()

    def _register_routes(self) -> None:
        self.router.add_api_route("", self.create, methods=["POST"], status_code=status.HTTP_201_CREATED)
        self.router.add_api_route("/", self.create, methods=["POST"], status_code=status.HTTP_201_CREATED)
        self.router.add_api_route("", self.get_all, methods=["GET"])
        self.router.add_api_route("/", self.get_all, methods=["GET"])
        self.router.add_api_route("/{order_id}", self.get_by_id, methods=["GET"])

    async def create(self, req: CreateOrderRequest):
        try:
            return await self.service.create_order(req)
        except ErrInvalidOrderInput as exc:
            return JSONResponse(status_code=status.HTTP_400_BAD_REQUEST, content={"error": str(exc)})
        except ErrProductNotFound as exc:
            return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"error": str(exc)})
        except (ErrInsufficientStock, ErrVersionConflict) as exc:
            return JSONResponse(status_code=status.HTTP_409_CONFLICT, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

    async def get_all(self, page: int = Query(1), limit: int = Query(10)):
        try:
            return await self.service.get_all_orders(page, limit)
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

    async def get_by_id(self, order_id: str):
        try:
            return await self.service.get_order_by_id(order_id)
        except ErrOrderNotFound as exc:
            return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

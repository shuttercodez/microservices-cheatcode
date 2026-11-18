from __future__ import annotations

from fastapi import APIRouter, Response, status
from fastapi.responses import JSONResponse

from app.product.model import (
    CreateProductRequest,
    ErrInvalidPayload,
    ErrNegativeStock,
    ErrProductNotFound,
    ErrVersionConflict,
    PatchStockRequest,
)
from app.product.service import Service


class Handler:
    """Manages HTTP requests and responses for product endpoints."""

    def __init__(self, service: Service) -> None:
        self.service = service
        self.router = APIRouter(prefix="/api/v1/products", tags=["products"])
        self._register_routes()

    def _register_routes(self) -> None:
        self.router.add_api_route("", self.create, methods=["POST"], status_code=status.HTTP_201_CREATED)
        self.router.add_api_route("/", self.create, methods=["POST"], status_code=status.HTTP_201_CREATED)
        self.router.add_api_route("", self.get_all, methods=["GET"])
        self.router.add_api_route("/", self.get_all, methods=["GET"])
        self.router.add_api_route("/{product_id}", self.get_by_id, methods=["GET"])
        self.router.add_api_route("/{product_id}", self.update, methods=["PUT"])
        self.router.add_api_route("/{product_id}/stock", self.patch_stock, methods=["PATCH"])
        self.router.add_api_route("/{product_id}", self.delete, methods=["DELETE"], status_code=status.HTTP_204_NO_CONTENT)

    async def create(self, req: CreateProductRequest):
        try:
            product = await self.service.create(req)
            return product
        except ErrInvalidPayload as exc:
            return JSONResponse(status_code=status.HTTP_400_BAD_REQUEST, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(status_code=status.HTTP_409_CONFLICT, content={"error": str(exc)})

    async def get_all(self):
        try:
            return await self.service.get_all()
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

    async def get_by_id(self, product_id: str):
        try:
            return await self.service.get_by_id(product_id)
        except ErrProductNotFound as exc:
            return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

    async def update(self, product_id: str, req: CreateProductRequest):
        try:
            return await self.service.update(product_id, req)
        except ErrProductNotFound as exc:
            return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"error": str(exc)})
        except ErrInvalidPayload as exc:
            return JSONResponse(status_code=status.HTTP_400_BAD_REQUEST, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

    async def patch_stock(self, product_id: str, req: PatchStockRequest):
        try:
            return await self.service.patch_stock(product_id, req)
        except ErrNegativeStock as exc:
            return JSONResponse(status_code=status.HTTP_400_BAD_REQUEST, content={"error": str(exc)})
        except ErrProductNotFound as exc:
            return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"error": str(exc)})
        except ErrVersionConflict as exc:
            return JSONResponse(status_code=status.HTTP_409_CONFLICT, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

    async def delete(self, product_id: str):
        try:
            await self.service.delete(product_id)
            return Response(status_code=status.HTTP_204_NO_CONTENT)
        except ErrProductNotFound as exc:
            return JSONResponse(status_code=status.HTTP_404_NOT_FOUND, content={"error": str(exc)})
        except Exception as exc:
            return JSONResponse(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                content={"error": str(exc)},
            )

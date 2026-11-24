import { apiRequest, DEFAULT_API_BASE_URL } from "@/lib/api-client";
import {
  CreateProductRequest,
  PaginatedProducts,
  Product,
} from "../types/product";

export const productsApi = {
  getAll(
    page = 1,
    limit = 10,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<PaginatedProducts> {
    return apiRequest<PaginatedProducts>(
      `/api/v1/products?page=${page}&limit=${limit}`,
      { cache: "no-store" },
      baseUrl
    );
  },

  create(
    req: CreateProductRequest,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<Product> {
    return apiRequest<Product>(
      "/api/v1/products",
      {
        method: "POST",
        body: JSON.stringify(req),
      },
      baseUrl
    );
  },

  update(
    id: string,
    req: CreateProductRequest,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<Product> {
    return apiRequest<Product>(
      `/api/v1/products/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(req),
      },
      baseUrl
    );
  },

  patchStock(
    id: string,
    stock: number,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<Product> {
    return apiRequest<Product>(
      `/api/v1/products/${id}/stock`,
      {
        method: "PATCH",
        body: JSON.stringify({ stock }),
      },
      baseUrl
    );
  },

  remove(id: string, baseUrl = DEFAULT_API_BASE_URL): Promise<void> {
    return apiRequest<void>(
      `/api/v1/products/${id}`,
      {
        method: "DELETE",
      },
      baseUrl
    );
  },
};

export interface Product {
  id: string;
  sku: string;
  name: string;
  price: number;
  stock: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface PaginatedProducts {
  data: Product[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface CreateProductRequest {
  sku: string;
  name: string;
  price: number;
  stock: number;
}

export interface Order {
  id: string;
  product_id: string;
  quantity: number;
  total_price: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface PaginatedOrders {
  data: Order[];
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface CreateOrderRequest {
  product_id: string;
  quantity: number;
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP error ${res.status}`);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return res.json();
}

export function useMicroserviceApi() {
  return {
    async checkHealth(baseUrl: string) {
      const [healthRes, readyRes] = await Promise.all([
        fetch(`${baseUrl}/healthz`).catch(() => null),
        fetch(`${baseUrl}/readyz`).catch(() => null),
      ]);
      return {
        alive: healthRes?.ok ?? false,
        ready: readyRes?.ok ?? false,
      };
    },

    async getProducts(
      page: number,
      limit: number,
      baseUrl: string
    ): Promise<PaginatedProducts> {
      const res = await fetch(
        `${baseUrl}/api/v1/products?page=${page}&limit=${limit}`,
        { cache: "no-store" }
      );
      return handleResponse<PaginatedProducts>(res);
    },

    async createProduct(
      req: CreateProductRequest,
      baseUrl: string
    ): Promise<Product> {
      const res = await fetch(`${baseUrl}/api/v1/products`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      return handleResponse<Product>(res);
    },

    async patchProductStock(
      id: string,
      stock: number,
      baseUrl: string
    ): Promise<Product> {
      const res = await fetch(`${baseUrl}/api/v1/products/${id}/stock`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ stock }),
      });
      return handleResponse<Product>(res);
    },

    async deleteProduct(id: string, baseUrl: string): Promise<void> {
      const res = await fetch(`${baseUrl}/api/v1/products/${id}`, {
        method: "DELETE",
      });
      return handleResponse<void>(res);
    },

    async getOrders(
      page: number,
      limit: number,
      baseUrl: string
    ): Promise<PaginatedOrders> {
      const res = await fetch(
        `${baseUrl}/api/v1/orders?page=${page}&limit=${limit}`,
        { cache: "no-store" }
      );
      return handleResponse<PaginatedOrders>(res);
    },

    async createOrder(
      req: CreateOrderRequest,
      baseUrl: string
    ): Promise<Order> {
      const res = await fetch(`${baseUrl}/api/v1/orders`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      return handleResponse<Order>(res);
    },
  };
}

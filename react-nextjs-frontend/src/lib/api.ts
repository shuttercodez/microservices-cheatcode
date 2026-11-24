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

const DEFAULT_API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

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

export const api = {
  async checkHealth(baseUrl = DEFAULT_API_BASE) {
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
    page = 1,
    limit = 10,
    baseUrl = DEFAULT_API_BASE
  ): Promise<PaginatedProducts> {
    const res = await fetch(
      `${baseUrl}/api/v1/products?page=${page}&limit=${limit}`,
      { cache: "no-store" }
    );
    return handleResponse<PaginatedProducts>(res);
  },

  async createProduct(
    req: CreateProductRequest,
    baseUrl = DEFAULT_API_BASE
  ): Promise<Product> {
    const res = await fetch(`${baseUrl}/api/v1/products`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
    return handleResponse<Product>(res);
  },

  async updateProduct(
    id: string,
    req: CreateProductRequest,
    baseUrl = DEFAULT_API_BASE
  ): Promise<Product> {
    const res = await fetch(`${baseUrl}/api/v1/products/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
    return handleResponse<Product>(res);
  },

  async patchProductStock(
    id: string,
    stock: number,
    baseUrl = DEFAULT_API_BASE
  ): Promise<Product> {
    const res = await fetch(`${baseUrl}/api/v1/products/${id}/stock`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ stock }),
    });
    return handleResponse<Product>(res);
  },

  async deleteProduct(id: string, baseUrl = DEFAULT_API_BASE): Promise<void> {
    const res = await fetch(`${baseUrl}/api/v1/products/${id}`, {
      method: "DELETE",
    });
    return handleResponse<void>(res);
  },

  async getOrders(
    page = 1,
    limit = 10,
    baseUrl = DEFAULT_API_BASE
  ): Promise<PaginatedOrders> {
    const res = await fetch(
      `${baseUrl}/api/v1/orders?page=${page}&limit=${limit}`,
      { cache: "no-store" }
    );
    return handleResponse<PaginatedOrders>(res);
  },

  async createOrder(
    req: CreateOrderRequest,
    baseUrl = DEFAULT_API_BASE
  ): Promise<Order> {
    const res = await fetch(`${baseUrl}/api/v1/orders`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    });
    return handleResponse<Order>(res);
  },
};

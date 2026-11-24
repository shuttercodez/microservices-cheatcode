import { apiRequest, DEFAULT_API_BASE_URL } from "@/lib/api-client";
import { CreateOrderRequest, Order, PaginatedOrders } from "../types/order";

export const ordersApi = {
  getAll(
    page = 1,
    limit = 10,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<PaginatedOrders> {
    return apiRequest<PaginatedOrders>(
      `/api/v1/orders?page=${page}&limit=${limit}`,
      { cache: "no-store" },
      baseUrl
    );
  },

  create(
    req: CreateOrderRequest,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<Order> {
    return apiRequest<Order>(
      "/api/v1/orders",
      {
        method: "POST",
        body: JSON.stringify(req),
      },
      baseUrl
    );
  },
};

import { PaginatedResponse } from "@/types/common";

export interface Order {
  id: string;
  product_id: string;
  quantity: number;
  total_price: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export type PaginatedOrders = PaginatedResponse<Order>;

export interface CreateOrderRequest {
  product_id: string;
  quantity: number;
}

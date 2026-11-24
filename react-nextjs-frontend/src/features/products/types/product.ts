import { PaginatedResponse } from "@/types/common";

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

export type PaginatedProducts = PaginatedResponse<Product>;

export interface CreateProductRequest {
  sku: string;
  name: string;
  price: number;
  stock: number;
}

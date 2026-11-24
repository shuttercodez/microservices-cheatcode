"use client";

import { useCallback, useState } from "react";
import { productsApi } from "../api/products-api";
import { CreateProductRequest, Product } from "../types/product";

export function useProducts(apiBaseUrl: string) {
  const [products, setProducts] = useState<Product[]>([]);
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(false);

  const fetchProducts = useCallback(async () => {
    setLoading(true);
    try {
      const res = await productsApi.getAll(page, limit, apiBaseUrl);
      setProducts(res.data || []);
      setTotal(res.total);
      setTotalPages(res.total_pages);
    } finally {
      setLoading(false);
    }
  }, [apiBaseUrl, page, limit]);

  const createProduct = async (req: CreateProductRequest) => {
    const created = await productsApi.create(req, apiBaseUrl);
    await fetchProducts();
    return created;
  };

  const patchStock = async (id: string, stock: number) => {
    const updated = await productsApi.patchStock(id, stock, apiBaseUrl);
    await fetchProducts();
    return updated;
  };

  const deleteProduct = async (id: string) => {
    await productsApi.remove(id, apiBaseUrl);
    await fetchProducts();
  };

  return {
    products,
    page,
    setPage,
    limit,
    total,
    totalPages,
    loading,
    fetchProducts,
    createProduct,
    patchStock,
    deleteProduct,
  };
}

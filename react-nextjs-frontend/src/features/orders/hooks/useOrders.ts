"use client";

import { useCallback, useState } from "react";
import { ordersApi } from "../api/orders-api";
import { CreateOrderRequest, Order } from "../types/order";

export function useOrders(apiBaseUrl: string) {
  const [orders, setOrders] = useState<Order[]>([]);
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(false);

  const fetchOrders = useCallback(async () => {
    setLoading(true);
    try {
      const res = await ordersApi.getAll(page, limit, apiBaseUrl);
      setOrders(res.data || []);
      setTotal(res.total);
      setTotalPages(res.total_pages);
    } finally {
      setLoading(false);
    }
  }, [apiBaseUrl, page, limit]);

  const createOrder = async (req: CreateOrderRequest) => {
    const created = await ordersApi.create(req, apiBaseUrl);
    await fetchOrders();
    return created;
  };

  return {
    orders,
    page,
    setPage,
    limit,
    total,
    totalPages,
    loading,
    fetchOrders,
    createOrder,
  };
}

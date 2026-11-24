"use client";

import React, { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { OrderForm } from "@/features/orders/components/OrderForm";
import { OrderTable } from "@/features/orders/components/OrderTable";
import { useOrders } from "@/features/orders/hooks/useOrders";
import { CreateOrderRequest } from "@/features/orders/types/order";
import { useProducts } from "@/features/products/hooks/useProducts";
import { useConsole } from "@/providers/ConsoleProvider";

function OrdersContent() {
  const searchParams = useSearchParams();
  const initialProductId = searchParams.get("productId") || "";

  const { apiBaseUrl, isAuthenticated, showSuccess, showError } = useConsole();
  const ordersState = useOrders(apiBaseUrl);
  const productsState = useProducts(apiBaseUrl);
  const [selectedProductId, setSelectedProductId] = useState(initialProductId);

  useEffect(() => {
    if (initialProductId) {
      setSelectedProductId(initialProductId);
    }
  }, [initialProductId]);

  useEffect(() => {
    if (!isAuthenticated) return;
    Promise.all([
      ordersState.fetchOrders(),
      productsState.fetchProducts(),
    ]).catch((err: unknown) => {
      showError(err instanceof Error ? err.message : "Failed to load orders");
    });
  }, [
    isAuthenticated,
    ordersState.fetchOrders,
    productsState.fetchProducts,
    showError,
  ]);

  if (!isAuthenticated) return null;

  const handleCreateOrder = async (req: CreateOrderRequest) => {
    try {
      const created = await ordersState.createOrder(req);
      await productsState.fetchProducts();
      showSuccess(
        `Order placed (#${created.id.slice(0, 8)}) — Total: $${created.total_price.toLocaleString()}`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to place order");
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <OrderForm
        products={productsState.products}
        selectedProductId={selectedProductId}
        onSelectProductId={setSelectedProductId}
        onCreate={handleCreateOrder}
      />
      <OrderTable
        orders={ordersState.orders}
        page={ordersState.page}
        totalPages={ordersState.totalPages}
        total={ordersState.total}
        onPageChange={ordersState.setPage}
      />
    </div>
  );
}

export default function OrdersPage() {
  return (
    <Suspense fallback={<div className="text-sm text-slate-500">Loading orders...</div>}>
      <OrdersContent />
    </Suspense>
  );
}

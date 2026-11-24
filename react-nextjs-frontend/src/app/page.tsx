"use client";

import React, { useCallback, useEffect, useState } from "react";
import { Navbar } from "@/components/layout/Navbar";
import { AlertBanner } from "@/components/ui/AlertBanner";
import { LoginCard } from "@/features/auth/components/LoginCard";
import { useAuth } from "@/features/auth/hooks/useAuth";
import {
  OAuthTokenRequest,
  RegisterUserRequest,
} from "@/features/auth/types/auth";
import { OrderForm } from "@/features/orders/components/OrderForm";
import { OrderTable } from "@/features/orders/components/OrderTable";
import { useOrders } from "@/features/orders/hooks/useOrders";
import { CreateOrderRequest } from "@/features/orders/types/order";
import { ProductForm } from "@/features/products/components/ProductForm";
import { ProductTable } from "@/features/products/components/ProductTable";
import { useProducts } from "@/features/products/hooks/useProducts";
import {
  CreateProductRequest,
  Product,
} from "@/features/products/types/product";
import { checkServiceHealth, DEFAULT_API_BASE_URL } from "@/lib/api-client";
import { HealthStatus } from "@/types/common";

export default function DashboardPage() {
  const [apiBaseUrl, setApiBaseUrl] = useState(DEFAULT_API_BASE_URL);
  const [health, setHealth] = useState<HealthStatus>({
    alive: false,
    ready: false,
  });
  const [activeTab, setActiveTab] = useState<"products" | "orders">("products");
  const [selectedProductId, setSelectedProductId] = useState("");

  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const authState = useAuth(apiBaseUrl);
  const productsState = useProducts(apiBaseUrl);
  const ordersState = useOrders(apiBaseUrl);

  const showSuccess = (msg: string) => {
    setSuccessMsg(msg);
    setErrorMsg(null);
    setTimeout(() => setSuccessMsg(null), 4000);
  };

  const showError = (msg: string) => {
    setErrorMsg(msg);
    setSuccessMsg(null);
  };

  const syncDashboard = useCallback(async () => {
    try {
      const status = await checkServiceHealth(apiBaseUrl);
      setHealth(status);
      if (authState.isAuthenticated) {
        await Promise.all([
          productsState.fetchProducts(),
          ordersState.fetchOrders(),
        ]);
      }
    } catch (err: unknown) {
      showError(
        err instanceof Error ? err.message : "Failed to connect to backend API"
      );
    }
  }, [
    apiBaseUrl,
    authState.isAuthenticated,
    productsState.fetchProducts,
    ordersState.fetchOrders,
  ]);

  useEffect(() => {
    syncDashboard();
  }, [syncDashboard]);

  const handleLogin = async (req: OAuthTokenRequest) => {
    try {
      const res = await authState.login(req);
      showSuccess(
        `Authenticated via OAuth 2.0 (${req.grant_type}) — Token expires in ${res.expires_in}s`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Authentication failed");
    }
  };

  const handleRegister = async (req: RegisterUserRequest) => {
    try {
      const user = await authState.register(req);
      await authState.login({
        grant_type: "password",
        username: req.username,
        password: req.password,
      });
      showSuccess(
        `Registered user "${user.username}" (${user.role}) in PostgreSQL and signed in!`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Registration failed");
    }
  };

  const handleCreateProduct = async (req: CreateProductRequest) => {
    try {
      const created = await productsState.createProduct(req);
      showSuccess(`Created product "${created.name}" (${created.sku})`);
    } catch (err: unknown) {
      showError(
        err instanceof Error ? err.message : "Failed to create product"
      );
    }
  };

  const handlePatchStock = async (productId: string, newStock: number) => {
    try {
      const updated = await productsState.patchStock(productId, newStock);
      showSuccess(
        `Updated stock for ${updated.sku} to ${updated.stock} (v${updated.version})`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to update stock");
    }
  };

  const handleDeleteProduct = async (product: Product) => {
    try {
      await productsState.deleteProduct(product.id);
      showSuccess(`Soft-deleted product ${product.sku}`);
    } catch (err: unknown) {
      showError(
        err instanceof Error ? err.message : "Failed to delete product"
      );
    }
  };

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
    <div className="max-w-6xl mx-auto px-4 py-8">
      <Navbar
        apiBaseUrl={apiBaseUrl}
        onApiBaseUrlChange={setApiBaseUrl}
        health={health}
        loading={productsState.loading || ordersState.loading}
        onRefresh={syncDashboard}
        isAuthenticated={authState.isAuthenticated}
        onLogout={() => {
          authState.logout();
          showSuccess("Logged out and cleared JWT token");
        }}
      />

      <AlertBanner
        type="error"
        message={errorMsg}
        onDismiss={() => setErrorMsg(null)}
      />
      <AlertBanner
        type="success"
        message={successMsg}
        onDismiss={() => setSuccessMsg(null)}
      />

      {!authState.isAuthenticated ? (
        <LoginCard
          onLogin={handleLogin}
          onRegister={handleRegister}
          enableRegister={authState.enableRegister}
          loading={authState.loading}
        />
      ) : (
        <>
          {/* Navigation Tabs */}
          <div className="flex gap-2 mb-6 border-b border-slate-200">
            <button
              onClick={() => setActiveTab("products")}
              className={`px-5 py-2.5 text-sm font-semibold border-b-2 -mb-px transition ${
                activeTab === "products"
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-slate-600 hover:text-slate-900"
              }`}
            >
              📦 Products ({productsState.total})
            </button>
            <button
              onClick={() => setActiveTab("orders")}
              className={`px-5 py-2.5 text-sm font-semibold border-b-2 -mb-px transition ${
                activeTab === "orders"
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-slate-600 hover:text-slate-900"
              }`}
            >
              🧾 Orders ({ordersState.total})
            </button>
          </div>

          {activeTab === "products" ? (
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <ProductForm onCreate={handleCreateProduct} />
              <ProductTable
                products={productsState.products}
                page={productsState.page}
                totalPages={productsState.totalPages}
                total={productsState.total}
                onPageChange={productsState.setPage}
                onPatchStock={handlePatchStock}
                onDelete={handleDeleteProduct}
                onSelectForOrder={(productId) => {
                  setSelectedProductId(productId);
                  setActiveTab("orders");
                }}
              />
            </div>
          ) : (
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
          )}
        </>
      )}
    </div>
  );
}

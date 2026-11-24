"use client";

import React, { useCallback, useEffect, useState } from "react";
import { api, Order, Product } from "@/lib/api";

export default function DashboardPage() {
  const [apiBaseUrl, setApiBaseUrl] = useState("http://localhost:8080");
  const [health, setHealth] = useState({ alive: false, ready: false });
  const [activeTab, setActiveTab] = useState<"products" | "orders">("products");

  // Products state
  const [products, setProducts] = useState<Product[]>([]);
  const [prodPage, setProdPage] = useState(1);
  const [prodLimit] = useState(10);
  const [prodTotal, setProdTotal] = useState(0);
  const [prodTotalPages, setProdTotalPages] = useState(0);

  // Create Product form
  const [sku, setSku] = useState("");
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");
  const [stock, setStock] = useState("");

  // Edit stock modal/inline state
  const [editingStockId, setEditingStockId] = useState<string | null>(null);
  const [newStockValue, setNewStockValue] = useState<string>("");

  // Orders state
  const [orders, setOrders] = useState<Order[]>([]);
  const [ordPage, setOrdPage] = useState(1);
  const [ordLimit] = useState(10);
  const [ordTotal, setOrdTotal] = useState(0);
  const [ordTotalPages, setOrdTotalPages] = useState(0);

  // Create Order form
  const [selectedProductId, setSelectedProductId] = useState("");
  const [orderQuantity, setOrderQuantity] = useState("1");

  // Status alerts
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const showSuccess = (msg: string) => {
    setSuccessMsg(msg);
    setErrorMsg(null);
    setTimeout(() => setSuccessMsg(null), 4000);
  };

  const showError = (msg: string) => {
    setErrorMsg(msg);
    setSuccessMsg(null);
  };

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const status = await api.checkHealth(apiBaseUrl);
      setHealth(status);

      const [prodRes, ordRes] = await Promise.all([
        api.getProducts(prodPage, prodLimit, apiBaseUrl),
        api.getOrders(ordPage, ordLimit, apiBaseUrl),
      ]);

      setProducts(prodRes.data || []);
      setProdTotal(prodRes.total);
      setProdTotalPages(prodRes.total_pages);

      setOrders(ordRes.data || []);
      setOrdTotal(ordRes.total);
      setOrdTotalPages(ordRes.total_pages);
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Failed to connect to backend API";
      showError(message);
    } finally {
      setLoading(false);
    }
  }, [apiBaseUrl, prodPage, prodLimit, ordPage, ordLimit]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const handleCreateProduct = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const created = await api.createProduct(
        {
          sku: sku.trim(),
          name: name.trim(),
          price: parseFloat(price) || 0,
          stock: parseInt(stock, 10) || 0,
        },
        apiBaseUrl
      );
      setSku("");
      setName("");
      setPrice("");
      setStock("");
      showSuccess(`Created product "${created.name}" (${created.sku})`);
      fetchAll();
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to create product");
    }
  };

  const handlePatchStock = async (productId: string) => {
    try {
      const parsed = parseInt(newStockValue, 10);
      if (isNaN(parsed)) return;
      const updated = await api.patchProductStock(
        productId,
        parsed,
        apiBaseUrl
      );
      setEditingStockId(null);
      showSuccess(
        `Updated stock for ${updated.sku} to ${updated.stock} (v${updated.version})`
      );
      fetchAll();
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to update stock");
    }
  };

  const handleDeleteProduct = async (product: Product) => {
    try {
      await api.deleteProduct(product.id, apiBaseUrl);
      showSuccess(`Soft-deleted product ${product.sku}`);
      fetchAll();
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to delete product");
    }
  };

  const handleCreateOrder = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const created = await api.createOrder(
        {
          product_id: selectedProductId.trim(),
          quantity: parseInt(orderQuantity, 10) || 1,
        },
        apiBaseUrl
      );
      setOrderQuantity("1");
      showSuccess(
        `Order placed (#${created.id.slice(0, 8)}) — Total: $${created.total_price.toLocaleString()}`
      );
      fetchAll();
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to place order");
    }
  };

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      {/* Header & Health Status */}
      <header className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 mb-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-0.5 text-xs font-semibold bg-blue-100 text-blue-800 rounded-full">
              React 19 + Next.js 15
            </span>
            <span
              className={`px-2.5 py-0.5 text-xs font-semibold rounded-full ${
                health.alive
                  ? "bg-emerald-100 text-emerald-800"
                  : "bg-rose-100 text-rose-800"
              }`}
            >
              /healthz: {health.alive ? "ALIVE" : "OFFLINE"}
            </span>
            <span
              className={`px-2.5 py-0.5 text-xs font-semibold rounded-full ${
                health.ready
                  ? "bg-emerald-100 text-emerald-800"
                  : "bg-amber-100 text-amber-800"
              }`}
            >
              /readyz: {health.ready ? "READY" : "NOT READY"}
            </span>
          </div>
          <h1 className="text-2xl font-bold text-slate-900 mt-2">
            Inventory &amp; Order Microservice Console
          </h1>
          <p className="text-sm text-slate-500">
            Connects seamlessly to Go (GORM), Go (SQLx), Rust (Axum), or Python
            (FastAPI) backends
          </p>
        </div>

        <div className="flex items-center gap-2">
          <input
            type="text"
            value={apiBaseUrl}
            onChange={(e) => setApiBaseUrl(e.target.value)}
            className="text-sm border border-slate-300 rounded-lg px-3 py-2 w-60 focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="http://localhost:8080"
          />
          <button
            onClick={fetchAll}
            disabled={loading}
            className="bg-slate-900 hover:bg-slate-800 text-white text-sm font-medium px-4 py-2 rounded-lg transition"
          >
            {loading ? "Syncing..." : "Refresh"}
          </button>
        </div>
      </header>

      {/* Alerts */}
      {errorMsg && (
        <div className="mb-6 bg-rose-50 border border-rose-200 text-rose-700 px-4 py-3 rounded-lg text-sm flex justify-between items-center">
          <span>⚠️ {errorMsg}</span>
          <button
            onClick={() => setErrorMsg(null)}
            className="text-rose-500 hover:text-rose-800 font-bold"
          >
            ✕
          </button>
        </div>
      )}
      {successMsg && (
        <div className="mb-6 bg-emerald-50 border border-emerald-200 text-emerald-800 px-4 py-3 rounded-lg text-sm flex justify-between items-center">
          <span>✅ {successMsg}</span>
          <button
            onClick={() => setSuccessMsg(null)}
            className="text-emerald-600 hover:text-emerald-900 font-bold"
          >
            ✕
          </button>
        </div>
      )}

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
          📦 Products ({prodTotal})
        </button>
        <button
          onClick={() => setActiveTab("orders")}
          className={`px-5 py-2.5 text-sm font-semibold border-b-2 -mb-px transition ${
            activeTab === "orders"
              ? "border-blue-600 text-blue-600"
              : "border-transparent text-slate-600 hover:text-slate-900"
          }`}
        >
          🧾 Orders ({ordTotal})
        </button>
      </div>

      {/* PRODUCTS TAB */}
      {activeTab === "products" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Create Product Card */}
          <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit">
            <h2 className="text-lg font-semibold text-slate-900 mb-4">
              Create New Product
            </h2>
            <form onSubmit={handleCreateProduct} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  SKU (Unique Code)
                </label>
                <input
                  type="text"
                  required
                  value={sku}
                  onChange={(e) => setSku(e.target.value)}
                  placeholder="e.g. IPHONE-16-PRO"
                  className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Product Name
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. iPhone 16 Pro 256GB"
                  className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-slate-700 mb-1">
                    Price
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    required
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    placeholder="39900"
                    className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-slate-700 mb-1">
                    Initial Stock
                  </label>
                  <input
                    type="number"
                    min="0"
                    required
                    value={stock}
                    onChange={(e) => setStock(e.target.value)}
                    placeholder="50"
                    className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
                  />
                </div>
              </div>
              <button
                type="submit"
                className="w-full bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium py-2.5 rounded-lg transition"
              >
                + Add Product
              </button>
            </form>
          </div>

          {/* Paginated Product Table */}
          <div className="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-200 flex justify-between items-center">
              <h2 className="text-lg font-semibold text-slate-900">
                Product Inventory (Redis Cached)
              </h2>
              <span className="text-xs text-slate-500">
                Page {prodPage} of {prodTotalPages || 1} ({prodTotal} items)
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-xs uppercase text-slate-500 border-b border-slate-200">
                  <tr>
                    <th className="px-6 py-3">SKU / Name</th>
                    <th className="px-4 py-3">Price</th>
                    <th className="px-4 py-3">Stock</th>
                    <th className="px-4 py-3">Version</th>
                    <th className="px-6 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {products.length === 0 ? (
                    <tr>
                      <td
                        colSpan={5}
                        className="px-6 py-8 text-center text-slate-400"
                      >
                        No products found. Create your first product on the
                        left.
                      </td>
                    </tr>
                  ) : (
                    products.map((p) => (
                      <tr key={p.id} className="hover:bg-slate-50">
                        <td className="px-6 py-3">
                          <div className="font-semibold text-slate-900">
                            {p.sku}
                          </div>
                          <div className="text-xs text-slate-500">{p.name}</div>
                          <div className="text-[11px] text-slate-400 font-mono">
                            {p.id}
                          </div>
                        </td>
                        <td className="px-4 py-3 font-medium">
                          ${p.price.toLocaleString()}
                        </td>
                        <td className="px-4 py-3">
                          {editingStockId === p.id ? (
                            <div className="flex items-center gap-1">
                              <input
                                type="number"
                                min="0"
                                value={newStockValue}
                                onChange={(e) =>
                                  setNewStockValue(e.target.value)
                                }
                                className="w-20 border border-slate-300 rounded px-2 py-1 text-xs"
                              />
                              <button
                                onClick={() => handlePatchStock(p.id)}
                                className="bg-emerald-600 text-white text-xs px-2 py-1 rounded"
                              >
                                Save
                              </button>
                              <button
                                onClick={() => setEditingStockId(null)}
                                className="text-slate-400 text-xs px-1"
                              >
                                ✕
                              </button>
                            </div>
                          ) : (
                            <span
                              className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                                p.stock > 5
                                  ? "bg-emerald-100 text-emerald-800"
                                  : p.stock > 0
                                  ? "bg-amber-100 text-amber-800"
                                  : "bg-rose-100 text-rose-800"
                              }`}
                            >
                              {p.stock} in stock
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-xs text-slate-500 font-mono">
                          v{p.version}
                        </td>
                        <td className="px-6 py-3 text-right space-x-2">
                          <button
                            onClick={() => {
                              setEditingStockId(p.id);
                              setNewStockValue(String(p.stock));
                            }}
                            className="text-xs text-blue-600 hover:text-blue-800 font-medium"
                          >
                            Patch Stock
                          </button>
                          <button
                            onClick={() => {
                              setSelectedProductId(p.id);
                              setActiveTab("orders");
                            }}
                            className="text-xs text-emerald-600 hover:text-emerald-800 font-medium"
                          >
                            Order
                          </button>
                          <button
                            onClick={() => handleDeleteProduct(p)}
                            className="text-xs text-rose-600 hover:text-rose-800 font-medium"
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            {/* Pagination Footer */}
            <div className="px-6 py-4 bg-slate-50 border-t border-slate-200 flex justify-between items-center">
              <button
                disabled={prodPage <= 1}
                onClick={() => setProdPage((prev) => Math.max(1, prev - 1))}
                className="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
              >
                ← Previous
              </button>
              <span className="text-xs text-slate-600">
                Page {prodPage} of {prodTotalPages || 1}
              </span>
              <button
                disabled={prodPage >= prodTotalPages}
                onClick={() => setProdPage((prev) => prev + 1)}
                className="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
              >
                Next →
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ORDERS TAB */}
      {activeTab === "orders" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Create Order Card */}
          <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit">
            <h2 className="text-lg font-semibold text-slate-900 mb-4">
              Place Customer Order
            </h2>
            <form onSubmit={handleCreateOrder} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Select Product (or enter UUID)
                </label>
                <select
                  value={selectedProductId}
                  onChange={(e) => setSelectedProductId(e.target.value)}
                  className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 mb-2 bg-white"
                >
                  <option value="">-- Choose from current page --</option>
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.sku} — {p.name} (${p.price}, Stock: {p.stock})
                    </option>
                  ))}
                </select>
                <input
                  type="text"
                  required
                  value={selectedProductId}
                  onChange={(e) => setSelectedProductId(e.target.value)}
                  placeholder="Product UUID"
                  className="w-full text-xs font-mono border border-slate-300 rounded-lg px-3 py-2"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Quantity
                </label>
                <input
                  type="number"
                  min="1"
                  required
                  value={orderQuantity}
                  onChange={(e) => setOrderQuantity(e.target.value)}
                  className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
                />
              </div>
              <button
                type="submit"
                className="w-full bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-medium py-2.5 rounded-lg transition"
              >
                Place Order &amp; Deduct Stock
              </button>
            </form>
          </div>

          {/* Paginated Order Table */}
          <div className="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-200 flex justify-between items-center">
              <h2 className="text-lg font-semibold text-slate-900">
                Customer Orders (Redis Cached)
              </h2>
              <span className="text-xs text-slate-500">
                Page {ordPage} of {ordTotalPages || 1} ({ordTotal} orders)
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-xs uppercase text-slate-500 border-b border-slate-200">
                  <tr>
                    <th className="px-6 py-3">Order ID</th>
                    <th className="px-4 py-3">Product ID</th>
                    <th className="px-4 py-3">Qty</th>
                    <th className="px-4 py-3">Total Price</th>
                    <th className="px-6 py-3">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {orders.length === 0 ? (
                    <tr>
                      <td
                        colSpan={5}
                        className="px-6 py-8 text-center text-slate-400"
                      >
                        No orders placed yet.
                      </td>
                    </tr>
                  ) : (
                    orders.map((o) => (
                      <tr key={o.id} className="hover:bg-slate-50">
                        <td className="px-6 py-3 font-mono text-xs text-slate-700">
                          {o.id}
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-slate-500">
                          {o.product_id}
                        </td>
                        <td className="px-4 py-3 font-semibold">{o.quantity}</td>
                        <td className="px-4 py-3 font-semibold text-emerald-700">
                          ${o.total_price.toLocaleString()}
                        </td>
                        <td className="px-6 py-3">
                          <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800">
                            {o.status}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            {/* Pagination Footer */}
            <div className="px-6 py-4 bg-slate-50 border-t border-slate-200 flex justify-between items-center">
              <button
                disabled={ordPage <= 1}
                onClick={() => setOrdPage((prev) => Math.max(1, prev - 1))}
                className="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
              >
                ← Previous
              </button>
              <span className="text-xs text-slate-600">
                Page {ordPage} of {ordTotalPages || 1}
              </span>
              <button
                disabled={ordPage >= ordTotalPages}
                onClick={() => setOrdPage((prev) => prev + 1)}
                className="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40"
              >
                Next →
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

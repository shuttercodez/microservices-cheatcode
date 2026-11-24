"use client";

import React, { useState } from "react";
import { Product } from "@/features/products/types/product";
import { CreateOrderRequest } from "../types/order";

interface OrderFormProps {
  products: Product[];
  selectedProductId: string;
  onSelectProductId: (id: string) => void;
  onCreate: (req: CreateOrderRequest) => Promise<void>;
}

export function OrderForm({
  products,
  selectedProductId,
  onSelectProductId,
  onCreate,
}: OrderFormProps) {
  const [quantity, setQuantity] = useState("1");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await onCreate({
        product_id: selectedProductId.trim(),
        quantity: parseInt(quantity, 10) || 1,
      });
      setQuantity("1");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit">
      <h2 className="text-lg font-semibold text-slate-900 mb-4">
        Place Customer Order
      </h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-xs font-medium text-slate-700 mb-1">
            Select Product (or enter UUID)
          </label>
          <select
            value={selectedProductId}
            onChange={(e) => onSelectProductId(e.target.value)}
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
            onChange={(e) => onSelectProductId(e.target.value)}
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
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
          />
        </div>
        <button
          type="submit"
          disabled={submitting}
          className="w-full bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-sm font-medium py-2.5 rounded-lg transition"
        >
          {submitting ? "Placing Order..." : "Place Order & Deduct Stock"}
        </button>
      </form>
    </div>
  );
}

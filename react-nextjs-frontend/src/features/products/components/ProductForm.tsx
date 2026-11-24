"use client";

import React, { useState } from "react";
import { CreateProductRequest } from "../types/product";

interface ProductFormProps {
  onCreate: (req: CreateProductRequest) => Promise<void>;
}

export function ProductForm({ onCreate }: ProductFormProps) {
  const [sku, setSku] = useState("");
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");
  const [stock, setStock] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await onCreate({
        sku: sku.trim(),
        name: name.trim(),
        price: parseFloat(price) || 0,
        stock: parseInt(stock, 10) || 0,
      });
      setSku("");
      setName("");
      setPrice("");
      setStock("");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit">
      <h2 className="text-lg font-semibold text-slate-900 mb-4">
        Create New Product
      </h2>
      <form onSubmit={handleSubmit} className="space-y-4">
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
          disabled={submitting}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium py-2.5 rounded-lg transition"
        >
          {submitting ? "Creating..." : "+ Add Product"}
        </button>
      </form>
    </div>
  );
}

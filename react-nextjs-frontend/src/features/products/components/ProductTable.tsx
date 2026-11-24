"use client";

import React, { useState } from "react";
import { Pagination } from "@/components/ui/Pagination";
import { Product } from "../types/product";

interface ProductTableProps {
  products: Product[];
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (page: number) => void;
  onPatchStock: (id: string, newStock: number) => Promise<void>;
  onDelete: (product: Product) => Promise<void>;
  onSelectForOrder: (productId: string) => void;
}

export function ProductTable({
  products,
  page,
  totalPages,
  total,
  onPageChange,
  onPatchStock,
  onDelete,
  onSelectForOrder,
}: ProductTableProps) {
  const [editingStockId, setEditingStockId] = useState<string | null>(null);
  const [newStockValue, setNewStockValue] = useState<string>("");

  const handleSaveStock = async (productId: string) => {
    const parsed = parseInt(newStockValue, 10);
    if (isNaN(parsed)) return;
    await onPatchStock(productId, parsed);
    setEditingStockId(null);
  };

  return (
    <div className="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
      <div className="px-6 py-4 border-b border-slate-200 flex justify-between items-center">
        <h2 className="text-lg font-semibold text-slate-900">
          Product Inventory (Redis Cached)
        </h2>
        <span className="text-xs text-slate-500">
          Page {page} of {totalPages || 1} ({total} items)
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
                  No products found. Create your first product on the left.
                </td>
              </tr>
            ) : (
              products.map((p) => (
                <tr key={p.id} className="hover:bg-slate-50">
                  <td className="px-6 py-3">
                    <div className="font-semibold text-slate-900">{p.sku}</div>
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
                          onChange={(e) => setNewStockValue(e.target.value)}
                          className="w-20 border border-slate-300 rounded px-2 py-1 text-xs"
                        />
                        <button
                          onClick={() => handleSaveStock(p.id)}
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
                      onClick={() => onSelectForOrder(p.id)}
                      className="text-xs text-emerald-600 hover:text-emerald-800 font-medium"
                    >
                      Order
                    </button>
                    <button
                      onClick={() => onDelete(p)}
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

      <Pagination
        page={page}
        totalPages={totalPages}
        onPageChange={onPageChange}
      />
    </div>
  );
}

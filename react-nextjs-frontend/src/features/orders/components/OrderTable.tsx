"use client";

import React from "react";
import { Pagination } from "@/components/ui/Pagination";
import { Order } from "../types/order";

interface OrderTableProps {
  orders: Order[];
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (page: number) => void;
}

export function OrderTable({
  orders,
  page,
  totalPages,
  total,
  onPageChange,
}: OrderTableProps) {
  return (
    <div className="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
      <div className="px-6 py-4 border-b border-slate-200 flex justify-between items-center">
        <h2 className="text-lg font-semibold text-slate-900">
          Customer Orders (Redis Cached)
        </h2>
        <span className="text-xs text-slate-500">
          Page {page} of {totalPages || 1} ({total} orders)
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

      <Pagination
        page={page}
        totalPages={totalPages}
        onPageChange={onPageChange}
      />
    </div>
  );
}

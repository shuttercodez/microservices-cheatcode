"use client";

import React from "react";

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (newPage: number) => void;
}

export function Pagination({
  page,
  totalPages,
  onPageChange,
}: PaginationProps) {
  return (
    <div className="px-6 py-4 bg-slate-50 border-t border-slate-200 flex justify-between items-center">
      <button
        disabled={page <= 1}
        onClick={() => onPageChange(Math.max(1, page - 1))}
        className="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50 transition"
      >
        ← Previous
      </button>
      <span className="text-xs text-slate-600">
        Page {page} of {totalPages || 1}
      </span>
      <button
        disabled={page >= (totalPages || 1)}
        onClick={() => onPageChange(page + 1)}
        className="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50 transition"
      >
        Next →
      </button>
    </div>
  );
}

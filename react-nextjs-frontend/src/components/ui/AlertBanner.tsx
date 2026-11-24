"use client";

import React from "react";

interface AlertBannerProps {
  type: "error" | "success";
  message: string | null;
  onDismiss: () => void;
}

export function AlertBanner({ type, message, onDismiss }: AlertBannerProps) {
  if (!message) return null;

  const isError = type === "error";

  return (
    <div
      className={`mb-6 px-4 py-3 rounded-lg text-sm flex justify-between items-center border ${
        isError
          ? "bg-rose-50 border-rose-200 text-rose-700"
          : "bg-emerald-50 border-emerald-200 text-emerald-800"
      }`}
    >
      <span>
        {isError ? "⚠️" : "✅"} {message}
      </span>
      <button
        onClick={onDismiss}
        className={`font-bold ${
          isError
            ? "text-rose-500 hover:text-rose-800"
            : "text-emerald-600 hover:text-emerald-900"
        }`}
      >
        ✕
      </button>
    </div>
  );
}

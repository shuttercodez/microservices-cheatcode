"use client";

import React from "react";
import { HealthStatus } from "@/types/common";

interface NavbarProps {
  apiBaseUrl: string;
  onApiBaseUrlChange: (url: string) => void;
  health: HealthStatus;
  loading: boolean;
  onRefresh: () => void;
}

export function Navbar({
  apiBaseUrl,
  onApiBaseUrlChange,
  health,
  loading,
  onRefresh,
}: NavbarProps) {
  return (
    <header className="bg-white rounded-xl shadow-sm border border-slate-200 p-6 mb-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4">
      <div>
        <div className="flex items-center gap-2">
          <span className="px-2.5 py-0.5 text-xs font-semibold bg-blue-100 text-blue-800 rounded-full">
            React 19 + Next.js 15 (Feature Architecture)
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
          onChange={(e) => onApiBaseUrlChange(e.target.value)}
          className="text-sm border border-slate-300 rounded-lg px-3 py-2 w-60 focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="http://localhost:8080"
        />
        <button
          onClick={onRefresh}
          disabled={loading}
          className="bg-slate-900 hover:bg-slate-800 text-white text-sm font-medium px-4 py-2 rounded-lg transition"
        >
          {loading ? "Syncing..." : "Refresh"}
        </button>
      </div>
    </header>
  );
}

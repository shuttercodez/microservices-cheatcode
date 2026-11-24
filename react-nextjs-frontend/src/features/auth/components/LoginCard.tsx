"use client";

import React, { useState } from "react";
import { OAuthTokenRequest, RegisterUserRequest } from "../types/auth";

type AuthMode = "password" | "register" | "client_credentials";

interface LoginCardProps {
  onLogin: (req: OAuthTokenRequest) => Promise<void>;
  onRegister: (req: RegisterUserRequest) => Promise<void>;
  enableRegister: boolean;
  loading: boolean;
}

export function LoginCard({
  onLogin,
  onRegister,
  enableRegister,
  loading,
}: LoginCardProps) {
  const [mode, setMode] = useState<AuthMode>("password");
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("adminpassword");
  const [role, setRole] = useState("user");
  const [clientId, setClientId] = useState("inventory-client");
  const [clientSecret, setClientSecret] = useState("inventory-secret");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (mode === "password") {
      await onLogin({
        grant_type: "password",
        username: username.trim(),
        password,
      });
    } else if (mode === "register") {
      await onRegister({
        username: username.trim(),
        password,
        role,
      });
    } else {
      await onLogin({
        grant_type: "client_credentials",
        client_id: clientId.trim(),
        client_secret: clientSecret,
      });
    }
  };

  return (
    <div className="max-w-md mx-auto bg-white rounded-xl shadow-sm border border-slate-200 p-6 my-8">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold text-slate-900">
          🔐 OAuth 2.0 + PostgreSQL Auth
        </h2>
        <span className="text-xs px-2 py-0.5 bg-slate-100 text-slate-600 rounded font-mono">
          {mode === "register" ? "POST /oauth/register" : "POST /oauth/token"}
        </span>
      </div>

      {/* Mode Switcher */}
      <div
        className={`grid ${
          enableRegister ? "grid-cols-3" : "grid-cols-2"
        } gap-1.5 p-1 bg-slate-100 rounded-lg mb-4 text-xs font-medium`}
      >
        <button
          type="button"
          onClick={() => setMode("password")}
          className={`py-1.5 rounded-md transition ${
            mode === "password"
              ? "bg-white text-slate-900 shadow-sm"
              : "text-slate-600"
          }`}
        >
          User Sign In
        </button>
        {enableRegister && (
          <button
            type="button"
            onClick={() => {
              setMode("register");
              if (username === "admin") {
                setUsername("");
                setPassword("");
              }
            }}
            className={`py-1.5 rounded-md transition ${
              mode === "register"
                ? "bg-white text-slate-900 shadow-sm"
                : "text-slate-600"
            }`}
          >
            Register DB User
          </button>
        )}
        <button
          type="button"
          onClick={() => setMode("client_credentials")}
          className={`py-1.5 rounded-md transition ${
            mode === "client_credentials"
              ? "bg-white text-slate-900 shadow-sm"
              : "text-slate-600"
          }`}
        >
          M2M Client
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {mode === "password" || mode === "register" ? (
          <>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">
                Username (stored in PostgreSQL `users` table)
              </label>
              <input
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter username"
                className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">
                Password (hashed with Bcrypt in DB)
              </label>
              <input
                type="password"
                required
                minLength={6}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Min 6 characters"
                className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
              />
            </div>
            {mode === "register" && (
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Role
                </label>
                <select
                  value={role}
                  onChange={(e) => setRole(e.target.value)}
                  className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 bg-white"
                >
                  <option value="user">user</option>
                  <option value="admin">admin</option>
                </select>
              </div>
            )}
          </>
        ) : (
          <>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">
                Client ID (stored in PostgreSQL `oauth_clients` table)
              </label>
              <input
                type="text"
                required
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
                className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">
                Client Secret (hashed with Bcrypt in DB)
              </label>
              <input
                type="password"
                required
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
                className="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 font-mono"
              />
            </div>
          </>
        )}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-semibold py-2.5 rounded-lg transition"
        >
          {loading
            ? "Processing..."
            : mode === "register"
            ? "Create Account & Sign In"
            : "Get OAuth 2.0 Access Token"}
        </button>
      </form>
    </div>
  );
}

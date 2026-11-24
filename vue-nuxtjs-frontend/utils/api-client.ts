import type { HealthStatus } from "~/types/common";

export const DEFAULT_API_BASE_URL = "http://localhost:8080";
const TOKEN_STORAGE_KEY = "microservice_jwt_token";
export const AUTH_EXPIRED_EVENT = "microservice:auth-expired";

export function isJwtExpired(token: string): boolean {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return true;
    const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const jsonPayload = atob(base64);
    const claims = JSON.parse(jsonPayload);
    if (typeof claims.exp === "number") {
      return Math.floor(Date.now() / 1000) >= claims.exp;
    }
    return false;
  } catch {
    return true;
  }
}

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  const token = localStorage.getItem(TOKEN_STORAGE_KEY);
  if (token && isJwtExpired(token)) {
    localStorage.removeItem(TOKEN_STORAGE_KEY);
    return null;
  }
  return token;
}

export function setAccessToken(token: string | null): void {
  if (typeof window === "undefined") return;
  if (token) {
    localStorage.setItem(TOKEN_STORAGE_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_STORAGE_KEY);
  }
}

export function notifySessionExpired(reason?: string): void {
  setAccessToken(null);
  if (typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent(AUTH_EXPIRED_EVENT, {
        detail:
          reason || "Your session has expired. Please sign in again.",
      })
    );
  }
}

export async function apiRequest<T>(
  endpoint: string,
  options?: RequestInit,
  baseUrl: string = DEFAULT_API_BASE_URL
): Promise<T> {
  const isProtectedEndpoint = endpoint.startsWith("/api/v1");
  const rawStoredToken =
    typeof window !== "undefined"
      ? localStorage.getItem(TOKEN_STORAGE_KEY)
      : null;

  if (isProtectedEndpoint && rawStoredToken && isJwtExpired(rawStoredToken)) {
    const msg = "Your session has expired. Please sign in again.";
    notifySessionExpired(msg);
    throw new Error(msg);
  }

  const token = getAccessToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };

  if (token && !headers["Authorization"]) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${baseUrl}${endpoint}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const errorMessage =
      body.error_description || body.error || `HTTP error ${res.status}`;

    // If a protected API returns 401 Unauthorized, automatically clear the token and redirect to Login
    if (res.status === 401 && isProtectedEndpoint) {
      const expiredMsg = "Your session has expired. Please sign in again.";
      notifySessionExpired(expiredMsg);
      throw new Error(expiredMsg);
    }

    throw new Error(errorMessage);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json();
}

export async function checkServiceHealth(
  baseUrl: string = DEFAULT_API_BASE_URL
): Promise<HealthStatus> {
  const [healthRes, readyRes] = await Promise.all([
    fetch(`${baseUrl}/healthz`).catch(() => null),
    fetch(`${baseUrl}/readyz`).catch(() => null),
  ]);

  return {
    alive: healthRes?.ok ?? false,
    ready: readyRes?.ok ?? false,
  };
}

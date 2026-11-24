import { HealthStatus } from "@/types/common";

export const DEFAULT_API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

const TOKEN_STORAGE_KEY = "microservice_jwt_token";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_STORAGE_KEY);
}

export function setAccessToken(token: string | null): void {
  if (typeof window === "undefined") return;
  if (token) {
    localStorage.setItem(TOKEN_STORAGE_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_STORAGE_KEY);
  }
}

export async function apiRequest<T>(
  endpoint: string,
  options?: RequestInit,
  baseUrl: string = DEFAULT_API_BASE_URL
): Promise<T> {
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
    throw new Error(
      body.error_description || body.error || `HTTP error ${res.status}`
    );
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

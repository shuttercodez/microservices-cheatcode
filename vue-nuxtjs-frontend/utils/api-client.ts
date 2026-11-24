import type { HealthStatus } from "~/types/common";

export const DEFAULT_API_BASE_URL = "http://localhost:8080";

export async function apiRequest<T>(
  endpoint: string,
  options?: RequestInit,
  baseUrl: string = DEFAULT_API_BASE_URL
): Promise<T> {
  const res = await fetch(`${baseUrl}${endpoint}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP error ${res.status}`);
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

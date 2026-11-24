import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  apiRequest,
  AUTH_EXPIRED_EVENT,
  getAccessToken,
  isJwtExpired,
  setAccessToken,
} from "./api-client";

function createTestJwt(expOffsetSeconds: number): string {
  const header = btoa(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = btoa(
    JSON.stringify({
      sub: "admin",
      exp: Math.floor(Date.now() / 1000) + expOffsetSeconds,
    })
  );
  return `${header}.${payload}.signature`;
}

describe("api-client JWT expiration & 401 interceptor", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("detects valid vs expired JWT payload timestamps", () => {
    const validToken = createTestJwt(3600);
    const expiredToken = createTestJwt(-60);

    expect(isJwtExpired(validToken)).toBe(false);
    expect(isJwtExpired(expiredToken)).toBe(true);
  });

  it("automatically clears expired JWT when reading from localStorage", () => {
    const expiredToken = createTestJwt(-30);
    setAccessToken(expiredToken);

    expect(getAccessToken()).toBeNull();
    expect(localStorage.getItem("microservice_jwt_token")).toBeNull();
  });

  it("dispatches AUTH_EXPIRED_EVENT and clears token when protected API returns 401", async () => {
    const validToken = createTestJwt(3600);
    setAccessToken(validToken);

    const expiredListener = vi.fn();
    window.addEventListener(AUTH_EXPIRED_EVENT, expiredListener);

    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ error: "invalid or expired JWT token" }),
    } as Response);

    await expect(
      apiRequest("/api/v1/products", {}, "http://localhost:8080")
    ).rejects.toThrow("Your session has expired. Please sign in again.");

    expect(expiredListener).toHaveBeenCalledTimes(1);
    expect(getAccessToken()).toBeNull();

    window.removeEventListener(AUTH_EXPIRED_EVENT, expiredListener);
  });
});

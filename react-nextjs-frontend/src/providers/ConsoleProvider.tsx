"use client";

import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { Navbar } from "@/components/layout/Navbar";
import { AlertBanner } from "@/components/ui/AlertBanner";
import { authApi } from "@/features/auth/api/auth-api";
import {
  OAuthTokenRequest,
  OAuthTokenResponse,
  RegisterUserRequest,
  UserResponse,
} from "@/features/auth/types/auth";
import {
  AUTH_EXPIRED_EVENT,
  checkServiceHealth,
  DEFAULT_API_BASE_URL,
  getAccessToken,
  isJwtExpired,
  setAccessToken,
} from "@/lib/api-client";
import { HealthStatus } from "@/types/common";

interface ConsoleContextValue {
  apiBaseUrl: string;
  setApiBaseUrl: (url: string) => void;
  health: HealthStatus;
  refreshHealth: () => Promise<void>;
  isAuthenticated: boolean;
  authReady: boolean;
  enableRegister: boolean;
  authLoading: boolean;
  login: (req: OAuthTokenRequest) => Promise<OAuthTokenResponse>;
  register: (req: RegisterUserRequest) => Promise<UserResponse>;
  logout: () => void;
  showSuccess: (msg: string) => void;
  showError: (msg: string) => void;
}

const ConsoleContext = createContext<ConsoleContextValue | null>(null);

export function ConsoleProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();

  const [apiBaseUrl, setApiBaseUrl] = useState(DEFAULT_API_BASE_URL);
  const [health, setHealth] = useState<HealthStatus>({
    alive: false,
    ready: false,
  });
  const [token, setTokenState] = useState<string | null>(null);
  const [authReady, setAuthReady] = useState(false);
  const [enableRegister, setEnableRegister] = useState(true);
  const [authLoading, setAuthLoading] = useState(false);

  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const showSuccess = useCallback((msg: string) => {
    setSuccessMsg(msg);
    setErrorMsg(null);
    setTimeout(() => setSuccessMsg(null), 4000);
  }, []);

  const showError = useCallback((msg: string) => {
    setErrorMsg(msg);
    setSuccessMsg(null);
  }, []);

  const refreshHealth = useCallback(async () => {
    const status = await checkServiceHealth(apiBaseUrl);
    setHealth(status);
  }, [apiBaseUrl]);

  // Initialize token & OAuth config
  useEffect(() => {
    const raw =
      typeof window !== "undefined"
        ? localStorage.getItem("microservice_jwt_token")
        : null;
    if (raw && isJwtExpired(raw)) {
      setAccessToken(null);
      setTokenState(null);
      showError("Your session has expired. Please sign in again.");
    } else {
      setTokenState(getAccessToken());
    }
    setAuthReady(true);
  }, [showError]);

  useEffect(() => {
    refreshHealth();
    authApi
      .getConfig(apiBaseUrl)
      .then((res) => setEnableRegister(res.enable_register))
      .catch(() => setEnableRegister(true));
  }, [apiBaseUrl, refreshHealth]);

  // Listen for 401 / expired token events from apiRequest()
  useEffect(() => {
    const handleExpired = (event: Event) => {
      const customEvent = event as CustomEvent<string>;
      setTokenState(null);
      showError(
        customEvent.detail || "Your session has expired. Please sign in again."
      );
      router.replace("/login");
    };

    window.addEventListener(AUTH_EXPIRED_EVENT, handleExpired);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleExpired);
  }, [router, showError]);

  // Route Guard: redirect unauthenticated users to /login, and authenticated users away from /login
  useEffect(() => {
    if (!authReady) return;
    const isAuth = Boolean(token);
    if (!isAuth && pathname !== "/login") {
      router.replace("/login");
    } else if (isAuth && (pathname === "/login" || pathname === "/")) {
      router.replace("/products");
    }
  }, [authReady, token, pathname, router]);

  const login = async (req: OAuthTokenRequest) => {
    setAuthLoading(true);
    try {
      const res = await authApi.requestToken(req, apiBaseUrl);
      setAccessToken(res.access_token);
      setTokenState(res.access_token);
      router.push("/products");
      return res;
    } finally {
      setAuthLoading(false);
    }
  };

  const register = async (req: RegisterUserRequest) => {
    setAuthLoading(true);
    try {
      return await authApi.registerUser(req, apiBaseUrl);
    } finally {
      setAuthLoading(false);
    }
  };

  const logout = () => {
    setAccessToken(null);
    setTokenState(null);
    showSuccess("Logged out and cleared JWT token");
    router.replace("/login");
  };

  return (
    <ConsoleContext.Provider
      value={{
        apiBaseUrl,
        setApiBaseUrl,
        health,
        refreshHealth,
        isAuthenticated: Boolean(token),
        authReady,
        enableRegister,
        authLoading,
        login,
        register,
        logout,
        showSuccess,
        showError,
      }}
    >
      <div className="max-w-6xl mx-auto px-4 py-8">
        <Navbar
          apiBaseUrl={apiBaseUrl}
          onApiBaseUrlChange={setApiBaseUrl}
          health={health}
          loading={authLoading}
          onRefresh={refreshHealth}
          isAuthenticated={Boolean(token)}
          onLogout={logout}
        />

        <AlertBanner
          type="error"
          message={errorMsg}
          onDismiss={() => setErrorMsg(null)}
        />
        <AlertBanner
          type="success"
          message={successMsg}
          onDismiss={() => setSuccessMsg(null)}
        />

        {authReady ? children : null}
      </div>
    </ConsoleContext.Provider>
  );
}

export function useConsole() {
  const ctx = useContext(ConsoleContext);
  if (!ctx) {
    throw new Error("useConsole must be used inside ConsoleProvider");
  }
  return ctx;
}

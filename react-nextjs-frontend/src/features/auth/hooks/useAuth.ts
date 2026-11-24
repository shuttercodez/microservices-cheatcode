"use client";

import { useEffect, useState } from "react";
import {
  AUTH_EXPIRED_EVENT,
  getAccessToken,
  isJwtExpired,
  notifySessionExpired,
  setAccessToken,
} from "@/lib/api-client";
import { authApi } from "../api/auth-api";
import { OAuthTokenRequest, RegisterUserRequest } from "../types/auth";

export function useAuth(
  apiBaseUrl: string,
  onSessionExpired?: (message: string) => void
) {
  const [token, setTokenState] = useState<string | null>(null);
  const [enableRegister, setEnableRegister] = useState<boolean>(true);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const raw =
      typeof window !== "undefined"
        ? localStorage.getItem("microservice_jwt_token")
        : null;
    if (raw && isJwtExpired(raw)) {
      notifySessionExpired("Your session has expired. Please sign in again.");
      setTokenState(null);
    } else {
      setTokenState(getAccessToken());
    }
  }, []);

  useEffect(() => {
    const handleExpired = (event: Event) => {
      const customEvent = event as CustomEvent<string>;
      setTokenState(null);
      if (onSessionExpired) {
        onSessionExpired(
          customEvent.detail ||
            "Your session has expired. Please sign in again."
        );
      }
    };

    window.addEventListener(AUTH_EXPIRED_EVENT, handleExpired);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleExpired);
  }, [onSessionExpired]);

  useEffect(() => {
    authApi
      .getConfig(apiBaseUrl)
      .then((res) => setEnableRegister(res.enable_register))
      .catch(() => setEnableRegister(true));
  }, [apiBaseUrl]);

  const login = async (req: OAuthTokenRequest) => {
    setLoading(true);
    try {
      const res = await authApi.requestToken(req, apiBaseUrl);
      setAccessToken(res.access_token);
      setTokenState(res.access_token);
      return res;
    } finally {
      setLoading(false);
    }
  };

  const register = async (req: RegisterUserRequest) => {
    setLoading(true);
    try {
      const user = await authApi.registerUser(req, apiBaseUrl);
      return user;
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    setAccessToken(null);
    setTokenState(null);
  };

  return {
    token,
    isAuthenticated: Boolean(token),
    enableRegister,
    loading,
    login,
    register,
    logout,
  };
}

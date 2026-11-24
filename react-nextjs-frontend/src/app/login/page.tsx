"use client";

import React from "react";
import { LoginCard } from "@/features/auth/components/LoginCard";
import {
  OAuthTokenRequest,
  RegisterUserRequest,
} from "@/features/auth/types/auth";
import { useConsole } from "@/providers/ConsoleProvider";

export default function LoginPage() {
  const { login, register, enableRegister, authLoading, showSuccess, showError } =
    useConsole();

  const handleLogin = async (req: OAuthTokenRequest) => {
    try {
      const res = await login(req);
      showSuccess(
        `Authenticated via OAuth 2.0 (${req.grant_type}) — Token expires in ${res.expires_in}s`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Authentication failed");
    }
  };

  const handleRegister = async (req: RegisterUserRequest) => {
    try {
      const user = await register(req);
      await login({
        grant_type: "password",
        username: req.username,
        password: req.password,
      });
      showSuccess(
        `Registered user "${user.username}" (${user.role}) in PostgreSQL and signed in!`
      );
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Registration failed");
    }
  };

  return (
    <LoginCard
      onLogin={handleLogin}
      onRegister={handleRegister}
      enableRegister={enableRegister}
      loading={authLoading}
    />
  );
}

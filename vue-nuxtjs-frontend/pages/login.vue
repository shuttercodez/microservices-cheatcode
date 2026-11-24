<script setup lang="ts">
import type { OAuthTokenRequest, RegisterUserRequest } from "~/types/auth";

const router = useRouter();
const {
  enableRegister,
  authLoading,
  login,
  register,
  showSuccess,
  showError,
} = useConsole();

async function handleLogin(req: OAuthTokenRequest) {
  try {
    const res = await login(req);
    showSuccess(
      `Authenticated via OAuth 2.0 (${req.grant_type}) — Token expires in ${res.expires_in}s`
    );
    await router.push("/products");
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Authentication failed");
  }
}

async function handleRegister(req: RegisterUserRequest) {
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
    await router.push("/products");
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Registration failed");
  }
}
</script>

<template>
  <AuthLoginCard
    :loading="authLoading"
    :enable-register="enableRegister"
    @login="handleLogin"
    @register="handleRegister"
  />
</template>

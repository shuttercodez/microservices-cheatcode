import { getAccessToken, isJwtExpired, setAccessToken } from "~/utils/api-client";

export default defineNuxtRouteMiddleware((to) => {
  if (import.meta.server) return;

  const { token, showError } = useConsole();
  const raw = localStorage.getItem("microservice_jwt_token");

  if (raw && isJwtExpired(raw)) {
    setAccessToken(null);
    token.value = null;
    showError("Your session has expired. Please sign in again.");
    if (to.path !== "/login") {
      return navigateTo("/login");
    }
    return;
  }

  const validToken = getAccessToken();
  token.value = validToken;

  if (!validToken && to.path !== "/login") {
    return navigateTo("/login");
  }

  if (validToken && (to.path === "/login" || to.path === "/")) {
    return navigateTo("/products");
  }
});

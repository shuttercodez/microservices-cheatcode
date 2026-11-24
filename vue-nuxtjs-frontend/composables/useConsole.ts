import type {
  AuthConfigResponse,
  OAuthTokenRequest,
  OAuthTokenResponse,
  RegisterUserRequest,
  UserResponse,
} from "~/types/auth";
import type { HealthStatus } from "~/types/common";
import {
  apiRequest,
  checkServiceHealth,
  DEFAULT_API_BASE_URL,
  getAccessToken,
  setAccessToken,
} from "~/utils/api-client";

export function useConsole() {
  const config = useRuntimeConfig();
  const apiBaseUrl = useState<string>(
    "apiBaseUrl",
    () => (config.public.apiBaseUrl as string) || DEFAULT_API_BASE_URL
  );
  const health = useState<HealthStatus>("health", () => ({
    alive: false,
    ready: false,
  }));
  const token = useState<string | null>("jwtToken", () => null);
  const enableRegister = useState<boolean>("enableRegister", () => true);
  const authLoading = useState<boolean>("authLoading", () => false);
  const errorMsg = useState<string | null>("errorMsg", () => null);
  const successMsg = useState<string | null>("successMsg", () => null);

  const isAuthenticated = computed(() => Boolean(token.value));

  function showSuccess(msg: string) {
    successMsg.value = msg;
    errorMsg.value = null;
    setTimeout(() => {
      successMsg.value = null;
    }, 4000);
  }

  function showError(msg: string) {
    errorMsg.value = msg;
    successMsg.value = null;
  }

  async function refreshHealth() {
    health.value = await checkServiceHealth(apiBaseUrl.value);
  }

  async function fetchAuthConfig() {
    try {
      const cfg = await apiRequest<AuthConfigResponse>(
        "/oauth/config",
        { cache: "no-store" },
        apiBaseUrl.value
      );
      enableRegister.value = cfg.enable_register;
    } catch {
      enableRegister.value = true;
    }
  }

  function syncTokenFromStorage() {
    token.value = getAccessToken();
  }

  async function login(req: OAuthTokenRequest): Promise<OAuthTokenResponse> {
    authLoading.value = true;
    try {
      const res = await apiRequest<OAuthTokenResponse>(
        "/oauth/token",
        {
          method: "POST",
          body: JSON.stringify(req),
        },
        apiBaseUrl.value
      );
      setAccessToken(res.access_token);
      token.value = res.access_token;
      return res;
    } finally {
      authLoading.value = false;
    }
  }

  async function register(req: RegisterUserRequest): Promise<UserResponse> {
    authLoading.value = true;
    try {
      return await apiRequest<UserResponse>(
        "/oauth/register",
        {
          method: "POST",
          body: JSON.stringify(req),
        },
        apiBaseUrl.value
      );
    } finally {
      authLoading.value = false;
    }
  }

  function logout() {
    setAccessToken(null);
    token.value = null;
  }

  return {
    apiBaseUrl,
    health,
    token,
    isAuthenticated,
    enableRegister,
    authLoading,
    errorMsg,
    successMsg,
    showSuccess,
    showError,
    refreshHealth,
    fetchAuthConfig,
    syncTokenFromStorage,
    login,
    register,
    logout,
  };
}

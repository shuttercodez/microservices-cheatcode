import { computed, onMounted, ref, watch, type Ref } from "vue";
import type {
  AuthConfigResponse,
  OAuthTokenRequest,
  OAuthTokenResponse,
  RegisterUserRequest,
  UserResponse,
} from "~/types/auth";
import { apiRequest, getAccessToken, setAccessToken } from "~/utils/api-client";

export function useAuth(apiBaseUrl: Ref<string>) {
  const token = ref<string | null>(null);
  const enableRegister = ref<boolean>(true);
  const loading = ref(false);

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

  onMounted(() => {
    token.value = getAccessToken();
    fetchAuthConfig();
  });

  watch(apiBaseUrl, () => {
    fetchAuthConfig();
  });

  const isAuthenticated = computed(() => Boolean(token.value));

  async function login(req: OAuthTokenRequest): Promise<OAuthTokenResponse> {
    loading.value = true;
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
      loading.value = false;
    }
  }

  async function register(req: RegisterUserRequest): Promise<UserResponse> {
    loading.value = true;
    try {
      const user = await apiRequest<UserResponse>(
        "/oauth/register",
        {
          method: "POST",
          body: JSON.stringify(req),
        },
        apiBaseUrl.value
      );
      return user;
    } finally {
      loading.value = false;
    }
  }

  function logout() {
    setAccessToken(null);
    token.value = null;
  }

  return {
    token,
    isAuthenticated,
    enableRegister,
    loading,
    login,
    register,
    logout,
  };
}

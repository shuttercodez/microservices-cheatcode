import { apiRequest, DEFAULT_API_BASE_URL } from "@/lib/api-client";
import {
  AuthConfigResponse,
  OAuthTokenRequest,
  OAuthTokenResponse,
  RegisterUserRequest,
  UserResponse,
} from "../types/auth";

export const authApi = {
  getConfig(baseUrl = DEFAULT_API_BASE_URL): Promise<AuthConfigResponse> {
    return apiRequest<AuthConfigResponse>(
      "/oauth/config",
      { cache: "no-store" },
      baseUrl
    );
  },

  requestToken(
    req: OAuthTokenRequest,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<OAuthTokenResponse> {
    return apiRequest<OAuthTokenResponse>(
      "/oauth/token",
      {
        method: "POST",
        body: JSON.stringify(req),
      },
      baseUrl
    );
  },

  registerUser(
    req: RegisterUserRequest,
    baseUrl = DEFAULT_API_BASE_URL
  ): Promise<UserResponse> {
    return apiRequest<UserResponse>(
      "/oauth/register",
      {
        method: "POST",
        body: JSON.stringify(req),
      },
      baseUrl
    );
  },
};

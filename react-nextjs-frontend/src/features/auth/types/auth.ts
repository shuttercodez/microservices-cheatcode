export type OAuthGrantType = "password" | "client_credentials";

export interface OAuthTokenRequest {
  grant_type: OAuthGrantType;
  username?: string;
  password?: string;
  client_id?: string;
  client_secret?: string;
  scope?: string;
}

export interface OAuthTokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  scope: string;
}

export interface RegisterUserRequest {
  username: string;
  password: string;
  role?: string;
}

export interface UserResponse {
  id: string;
  username: string;
  role: string;
  created_at: string;
  updated_at: string;
}

export interface AuthConfigResponse {
  enable_register: boolean;
}

import { apiClient, type ApiClient } from "@/lib/api-client";
import type { AdminUser, AuthResponse, LoginRequest, StatusResponse } from "@/types/api";

export function login(payload: LoginRequest, client: ApiClient = apiClient) {
  return client.post<AuthResponse>("/auth/login", payload);
}

export function logout(client: ApiClient = apiClient) {
  return client.post<StatusResponse>("/auth/logout");
}

export function getCurrentUser(client: ApiClient = apiClient) {
  return client.get<AdminUser>("/auth/me");
}

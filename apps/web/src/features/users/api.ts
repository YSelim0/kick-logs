import { apiClient, type ApiClient } from "@/lib/api-client";
import type { AdminUser, CreateAdminUserRequest } from "@/types/api";

export function listAdminUsers(client: ApiClient = apiClient) {
  return client.get<AdminUser[]>("/admin/users");
}

export function createAdminUser(payload: CreateAdminUserRequest, client: ApiClient = apiClient) {
  return client.post<AdminUser>("/admin/users", payload);
}

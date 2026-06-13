import { apiClient, type ApiClient } from "@/lib/api-client";
import type { CreateUserRequestRequest, CreateUserRequestResponse } from "@/types/api";

export function createUserRequest(
  payload: CreateUserRequestRequest,
  client: ApiClient = apiClient
) {
  return client.post<CreateUserRequestResponse>("/requests", payload);
}

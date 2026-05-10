import { apiClient, type ApiClient } from "@/lib/api-client";
import type { HealthResponse } from "@/types/api";

export function getHealth(client: ApiClient = apiClient) {
  return client.get<HealthResponse>("/health");
}

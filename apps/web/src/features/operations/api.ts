import { apiClient, type ApiClient } from "@/lib/api-client";
import type { OperationsSummary } from "@/types/api";

export function getOperationsSummary(client: ApiClient = apiClient) {
  return client.get<OperationsSummary>("/admin/operations/summary");
}

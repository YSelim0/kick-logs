import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  DataCleanupConfirmRequest,
  DataCleanupPreview,
  DataCleanupRequest,
  DataCleanupResult,
  DataManagementSummary,
  RetentionSettings,
  UpdateRetentionSettingsRequest
} from "@/types/api";

export function getDataManagementSummary(client: ApiClient = apiClient) {
  return client.get<DataManagementSummary>("/admin/data-management/summary");
}

export function updateRetentionSettings(
  body: UpdateRetentionSettingsRequest,
  client: ApiClient = apiClient
) {
  return client.request<RetentionSettings>("/admin/data-management/retention-settings", {
    method: "PUT",
    body
  });
}

export function previewDataCleanup(body: DataCleanupRequest, client: ApiClient = apiClient) {
  return client.post<DataCleanupPreview>("/admin/data-management/cleanup/preview", body);
}

export function confirmDataCleanup(body: DataCleanupConfirmRequest, client: ApiClient = apiClient) {
  return client.post<DataCleanupResult>("/admin/data-management/cleanup/confirm", body);
}

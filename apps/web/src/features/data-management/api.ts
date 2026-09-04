import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  DataCleanupConfirmRequest,
  DataCleanupPreview,
  DataCleanupRequest,
  DataCleanupResult,
  DataManagementSummary,
  MessageImportPreview,
  MessageImportResult,
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

export function previewMessageImport(
  file: File,
  limit: number | null,
  client: ApiClient = apiClient
) {
  return client.post<MessageImportPreview>(
    "/admin/data-management/import/preview",
    buildImportForm(file, limit)
  );
}

export function confirmMessageImport(
  file: File,
  limit: number | null,
  confirmationText: string,
  client: ApiClient = apiClient
) {
  const form = buildImportForm(file, limit);
  form.append("confirmation_text", confirmationText);
  return client.post<MessageImportResult>("/admin/data-management/import/confirm", form);
}

function buildImportForm(file: File, limit: number | null) {
  const form = new FormData();
  form.append("file", file);
  if (limit !== null && limit > 0) {
    form.append("limit", String(limit));
  }
  return form;
}

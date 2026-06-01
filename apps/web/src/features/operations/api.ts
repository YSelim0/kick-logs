import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  FailedEventsActionResponse,
  FailedRawEventsResponse,
  OperationsSummary,
  WebhookHealth
} from "@/types/api";

export function getOperationsSummary(client: ApiClient = apiClient) {
  return client.get<OperationsSummary>("/admin/operations/summary");
}

export function getFailedEvents(client: ApiClient = apiClient) {
  return client.get<FailedRawEventsResponse>("/admin/operations/failed-events");
}

export function retryFailedEvents(client: ApiClient = apiClient) {
  return client.post<FailedEventsActionResponse>("/admin/operations/failed-events/retry", {});
}

export function clearFailedEvents(client: ApiClient = apiClient) {
  return client.post<FailedEventsActionResponse>("/admin/operations/failed-events/clear", {});
}

export function getWebhookHealth(client: ApiClient = apiClient) {
  return client.get<WebhookHealth>("/admin/webhooks/health");
}

export function triggerWebhookSync(client: ApiClient = apiClient) {
  return client.post<{ status: string }>("/admin/webhooks/sync", {});
}

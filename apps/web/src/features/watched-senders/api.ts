import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  AddWatchedSenderRequest,
  NotificationSettings,
  UpdateNotificationSettingsRequest,
  WatchedSender
} from "@/types/api";

export function listWatchedSenders(client: ApiClient = apiClient) {
  return client.get<WatchedSender[]>("/admin/watched-senders");
}

export function addWatchedSender(payload: AddWatchedSenderRequest, client: ApiClient = apiClient) {
  return client.post<WatchedSender>("/admin/watched-senders", payload);
}

export function removeWatchedSender(senderId: number, client: ApiClient = apiClient) {
  return client.delete<{ status: string }>(`/admin/watched-senders/${senderId}`);
}

export function getNotificationSettings(client: ApiClient = apiClient) {
  return client.get<NotificationSettings>("/admin/notification-settings");
}

export function updateNotificationSettings(
  payload: UpdateNotificationSettingsRequest,
  client: ApiClient = apiClient
) {
  return client.request<NotificationSettings>("/admin/notification-settings", {
    method: "PUT",
    body: payload
  });
}

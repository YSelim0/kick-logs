import { apiClient, type ApiClient } from "@/lib/api-client";
import type { AddWatchedSenderRequest, WatchedSender } from "@/types/api";

export function listWatchedSenders(client: ApiClient = apiClient) {
  return client.get<WatchedSender[]>("/admin/watched-senders");
}

export function addWatchedSender(payload: AddWatchedSenderRequest, client: ApiClient = apiClient) {
  return client.post<WatchedSender>("/admin/watched-senders", payload);
}

export function removeWatchedSender(senderId: number, client: ApiClient = apiClient) {
  return client.delete<{ status: string }>(`/admin/watched-senders/${senderId}`);
}

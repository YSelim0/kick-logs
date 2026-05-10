import { apiClient, type ApiClient } from "@/lib/api-client";
import type { AddChannelRequest, Channel } from "@/types/api";

export function listChannels(client: ApiClient = apiClient) {
  return client.get<Channel[]>("/admin/channels");
}

export function addChannel(payload: AddChannelRequest, client: ApiClient = apiClient) {
  return client.post<Channel>("/admin/channels", payload);
}

export function removeChannel(channelId: number, client: ApiClient = apiClient) {
  return client.delete<Channel>(`/admin/channels/${channelId}`);
}

import { apiClient, type ApiClient } from "@/lib/api-client";
import type { ChannelProfile } from "@/types/api";

export function getChannelProfile(slug: string, client: ApiClient = apiClient) {
  return client.get<ChannelProfile>(`/channels/${encodeURIComponent(slug)}/analytics`);
}

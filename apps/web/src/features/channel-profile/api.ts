import { apiClient, type ApiClient } from "@/lib/api-client";
import type { ChannelProfile, ChannelSubscriptionSummary } from "@/types/api";

export function getChannelProfile(slug: string, client: ApiClient = apiClient) {
  return client.get<ChannelProfile>(`/channels/${encodeURIComponent(slug)}/analytics`);
}

export function getChannelSubscriptionSummary(slug: string, client: ApiClient = apiClient) {
  return client.get<ChannelSubscriptionSummary>(
    `/channels/${encodeURIComponent(slug)}/subscription-summary`
  );
}

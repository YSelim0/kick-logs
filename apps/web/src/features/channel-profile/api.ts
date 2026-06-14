import { apiClient, type ApiClient } from "@/lib/api-client";
import { API_BASE_URL } from "@/lib/constants";
import type {
  ChannelProfile,
  ChannelSubscriberExportFormat,
  ChannelSubscribersResponse,
  ChannelSubscriptionSummary
} from "@/types/api";

export type ChannelSubscribersParams = {
  limit?: number;
  offset?: number;
  gift_only?: boolean;
};

export function getChannelProfile(slug: string, client: ApiClient = apiClient) {
  return client.get<ChannelProfile>(`/channels/${encodeURIComponent(slug)}/analytics`);
}

export function getChannelSubscriptionSummary(slug: string, client: ApiClient = apiClient) {
  return client.get<ChannelSubscriptionSummary>(
    `/channels/${encodeURIComponent(slug)}/subscription-summary`
  );
}

export function getChannelSubscribers(
  slug: string,
  params: ChannelSubscribersParams = {},
  client: ApiClient = apiClient
) {
  return client.get<ChannelSubscribersResponse>(
    `/channels/${encodeURIComponent(slug)}/subscribers`,
    {
      limit: params.limit,
      offset: params.offset,
      gift_only: params.gift_only
    }
  );
}

export function buildChannelSubscribersExportUrl(
  slug: string,
  giftOnly: boolean,
  format: ChannelSubscriberExportFormat,
  baseUrl = API_BASE_URL
) {
  const url = new URL(
    `${baseUrl.replace(/\/+$/, "")}/channels/${encodeURIComponent(slug)}/subscribers/export`
  );
  url.searchParams.set("format", format);
  if (giftOnly) {
    url.searchParams.set("gift_only", "true");
  }
  return url.toString();
}

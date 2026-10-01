import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  AnalyticsOverview,
  MessageVolumePoint,
  TopChannelAnalytics,
  TopEmoteAnalytics,
  TopSenderAnalytics
} from "@/types/api";

export type HomepageSnapshot = {
  status: "ready";
  as_of: string;
  start: string;
  end: string;
  timezone: "UTC";
  stale: boolean;
  overview: AnalyticsOverview;
  message_volume: MessageVolumePoint[];
  top_channels: TopChannelAnalytics[];
  top_senders: TopSenderAnalytics[];
  top_emotes: TopEmoteAnalytics[];
};

export type HomepageResponse =
  | HomepageSnapshot
  | { status: "initializing"; retry_after_seconds: number };

export function getHomepage(
  signal?: AbortSignal,
  client: ApiClient = apiClient
): Promise<HomepageResponse> {
  return client.get<HomepageResponse>("/analytics/homepage", undefined, {
    cache: "no-store",
    signal
  });
}

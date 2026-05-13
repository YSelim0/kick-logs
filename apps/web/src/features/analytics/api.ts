import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  AnalyticsBucket,
  AnalyticsOverview,
  MessageVolumeResponse,
  TopChannelsResponse,
  TopEmotesResponse,
  TopSendersResponse
} from "@/types/api";

export type AnalyticsQueryParams = {
  start?: string;
  end?: string;
  channel?: string;
  sender?: string;
  limit?: number;
  bucket?: AnalyticsBucket;
};

export function getAnalyticsOverview(
  params: AnalyticsQueryParams = {},
  client: ApiClient = apiClient
) {
  return client.get<AnalyticsOverview>("/analytics/overview", buildAnalyticsQuery(params));
}

export function getMessageVolume(params: AnalyticsQueryParams = {}, client: ApiClient = apiClient) {
  return client.get<MessageVolumeResponse>(
    "/analytics/message-volume",
    buildAnalyticsQuery(params)
  );
}

export function getTopSenders(params: AnalyticsQueryParams = {}, client: ApiClient = apiClient) {
  return client.get<TopSendersResponse>("/analytics/top-senders", buildAnalyticsQuery(params));
}

export function getTopChannels(params: AnalyticsQueryParams = {}, client: ApiClient = apiClient) {
  return client.get<TopChannelsResponse>("/analytics/top-channels", buildAnalyticsQuery(params));
}

export function getTopEmotes(params: AnalyticsQueryParams = {}, client: ApiClient = apiClient) {
  return client.get<TopEmotesResponse>("/analytics/top-emotes", buildAnalyticsQuery(params));
}

function buildAnalyticsQuery(params: AnalyticsQueryParams) {
  return {
    start: params.start,
    end: params.end,
    channel: params.channel,
    sender: params.sender,
    limit: params.limit,
    bucket: params.bucket
  };
}

import { API_BASE_URL, DEFAULT_MESSAGE_LIMIT } from "@/lib/constants";
import { apiClient, type ApiClient } from "@/lib/api-client";
import type { MessageExportFormat, MessageSearchParams, MessageSearchResponse } from "@/types/api";

export function searchMessages(params: MessageSearchParams = {}, client: ApiClient = apiClient) {
  return client.get<MessageSearchResponse>("/messages", {
    limit: params.limit ?? DEFAULT_MESSAGE_LIMIT,
    sender: params.sender,
    channel: params.channel,
    q: params.q,
    start: params.start,
    end: params.end,
    reply_only: params.reply_only,
    emote_only: params.emote_only,
    cursor: params.cursor
  });
}

export function buildMessageExportUrl(
  params: MessageSearchParams,
  format: MessageExportFormat,
  baseUrl = API_BASE_URL
) {
  const url = new URL(`${baseUrl.replace(/\/+$/, "")}/messages/export`);
  const query: MessageSearchParams & { format: MessageExportFormat } = { ...params, format };

  delete query.cursor;

  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }

  return url.toString();
}

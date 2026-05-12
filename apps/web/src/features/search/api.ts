import { DEFAULT_MESSAGE_LIMIT } from "@/lib/constants";
import { apiClient, type ApiClient } from "@/lib/api-client";
import type { MessageSearchParams, MessageSearchResponse } from "@/types/api";

export function searchMessages(params: MessageSearchParams = {}, client: ApiClient = apiClient) {
  return client.get<MessageSearchResponse>("/messages", {
    limit: params.limit ?? DEFAULT_MESSAGE_LIMIT,
    sender: params.sender,
    channel: params.channel,
    q: params.q,
    start: params.start,
    end: params.end,
    cursor: params.cursor
  });
}

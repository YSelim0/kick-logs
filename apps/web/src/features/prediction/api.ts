import { apiClient, type ApiClient } from "@/lib/api-client";
import type { Prediction } from "@/types/api";

export function getPrediction(slug: string, client: ApiClient = apiClient) {
  return client.get<Prediction>(`/channels/${encodeURIComponent(slug)}/prediction`);
}

import { apiClient, type ApiClient } from "@/lib/api-client";
import type { UserProfile } from "@/types/api";

export function getUserProfile(slug: string, client: ApiClient = apiClient) {
  return client.get<UserProfile>(`/users/${encodeURIComponent(slug)}/analytics`);
}

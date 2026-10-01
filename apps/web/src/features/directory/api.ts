import { apiClient, type ApiClient } from "@/lib/api-client";
import type { DirectoryPage, DirectoryQuery } from "@/types/directory";

export function getDirectoryUsers(
  query: DirectoryQuery,
  options: Pick<RequestInit, "signal"> = {},
  client: ApiClient = apiClient
) {
  return client.get<DirectoryPage>("/directory/users", query, options);
}

export function getDirectoryChannels(
  query: DirectoryQuery,
  options: Pick<RequestInit, "signal"> = {},
  client: ApiClient = apiClient
) {
  return client.get<DirectoryPage>("/directory/channels", query, options);
}

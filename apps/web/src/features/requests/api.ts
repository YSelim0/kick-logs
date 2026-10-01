import { apiClient, type ApiClient } from "@/lib/api-client";
import type {
  AddUserRequestNoteRequest,
  CreateUserRequestRequest,
  CreateUserRequestResponse,
  UpdateUserRequestStatusRequest,
  UserRequestDetailResponse,
  UserRequestListParams,
  UserRequestsResponse
} from "@/types/api";

export function createUserRequest(
  payload: CreateUserRequestRequest,
  client: ApiClient = apiClient
) {
  return client.post<CreateUserRequestResponse>("/requests", payload);
}

export function listUserRequests(
  params: UserRequestListParams = {},
  client: ApiClient = apiClient
) {
  return client.get<UserRequestsResponse>("/admin/requests", params);
}

export function getUserRequest(requestID: string, client: ApiClient = apiClient) {
  return client.get<UserRequestDetailResponse>(`/admin/requests/${encodeURIComponent(requestID)}`);
}

export function updateUserRequestStatus(
  requestID: string,
  payload: UpdateUserRequestStatusRequest,
  client: ApiClient = apiClient
) {
  return client.post<UserRequestDetailResponse>(
    `/admin/requests/${encodeURIComponent(requestID)}/status`,
    payload
  );
}

export function addUserRequestNote(
  requestID: string,
  payload: AddUserRequestNoteRequest,
  client: ApiClient = apiClient
) {
  return client.post<UserRequestDetailResponse>(
    `/admin/requests/${encodeURIComponent(requestID)}/notes`,
    payload
  );
}

export function archiveUserRequest(requestID: string, client: ApiClient = apiClient) {
  return client.post<UserRequestDetailResponse>(
    `/admin/requests/${encodeURIComponent(requestID)}/archive`
  );
}

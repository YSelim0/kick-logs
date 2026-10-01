import { ApiClientError } from "@/lib/api-client";

export type UiOperation =
  | "login"
  | "session"
  | "logout"
  | "search"
  | "directory"
  | "profile"
  | "prediction"
  | "requestSubmit"
  | "adminRead"
  | "adminMutation"
  | "cleanup";
export type UiErrorKey =
  | "invalidCredentials"
  | "sessionExpired"
  | "forbidden"
  | "notFound"
  | "conflict"
  | "invalidInput"
  | "rateLimited"
  | "network"
  | "unavailable";

export function getUiErrorKey(error: unknown, operation: UiOperation): UiErrorKey {
  if (error instanceof ApiClientError) {
    switch (error.status) {
      case 401:
        return operation === "login" ? "invalidCredentials" : "sessionExpired";
      case 403:
        return "forbidden";
      case 404:
        return "notFound";
      case 409:
        return "conflict";
      case 400:
      case 422:
        return "invalidInput";
      case 429:
        return "rateLimited";
      default:
        return "unavailable";
    }
  }
  return error instanceof TypeError ? "network" : "unavailable";
}

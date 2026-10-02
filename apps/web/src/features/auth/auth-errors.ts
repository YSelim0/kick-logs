import { ApiClientError } from "@/lib/api-client";

export function isUnauthorizedError(error: unknown) {
  return error instanceof ApiClientError && error.status === 401;
}

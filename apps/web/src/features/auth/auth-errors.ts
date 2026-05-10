import { ApiClientError } from "@/lib/api-client";

export function isUnauthorizedError(error: unknown) {
  return error instanceof ApiClientError && error.status === 401;
}

export function getAuthErrorMessage(error: unknown) {
  if (isUnauthorizedError(error)) {
    return "E-posta veya parola hatalı.";
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "İşlem tamamlanırken bir hata oluştu.";
}

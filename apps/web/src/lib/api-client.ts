import { API_BASE_URL } from "@/lib/constants";
import type { ApiErrorBody } from "@/types/api";

type QueryValue = string | number | boolean | null | undefined;
type QueryParams = Record<string, QueryValue>;
type ApiFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

type ApiClientOptions = {
  baseUrl?: string;
  fetcher?: ApiFetch;
};

type RequestOptions = Omit<RequestInit, "body"> & {
  query?: QueryParams;
  body?: unknown;
};

export class ApiClientError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(status: number, body: unknown) {
    super(resolveErrorMessage(status, body));
    this.name = "ApiClientError";
    this.status = status;
    this.body = body;
  }
}

export function createApiClient(options: ApiClientOptions = {}) {
  const baseUrl = trimTrailingSlash(options.baseUrl ?? API_BASE_URL);
  const fetcher = options.fetcher ?? fetch;

  async function request<TResponse>(path: string, options: RequestOptions = {}) {
    const headers = new Headers(options.headers);
    const hasBody = options.body !== undefined;

    if (hasBody && !headers.has("content-type")) {
      headers.set("content-type", "application/json");
    }

    const response = await fetcher(buildUrl(baseUrl, path, options.query), {
      ...options,
      headers,
      credentials: options.credentials ?? "include",
      body: hasBody ? JSON.stringify(options.body) : undefined
    });

    if (!response.ok) {
      throw new ApiClientError(response.status, await readResponseBody(response));
    }

    return (await readResponseBody(response)) as TResponse;
  }

  return {
    get<TResponse>(path: string, query?: QueryParams, options?: RequestInit) {
      return request<TResponse>(path, { ...options, method: "GET", query });
    },
    post<TResponse>(path: string, body?: unknown, options?: RequestInit) {
      return request<TResponse>(path, { ...options, method: "POST", body });
    },
    delete<TResponse>(path: string, options?: RequestInit) {
      return request<TResponse>(path, { ...options, method: "DELETE" });
    },
    request
  };
}

export const apiClient = createApiClient();

export type ApiClient = ReturnType<typeof createApiClient>;

function buildUrl(baseUrl: string, path: string, query?: QueryParams) {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const url = `${baseUrl}${normalizedPath}`;
  const searchParams = new URLSearchParams();

  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== null && value !== "") {
        searchParams.set(key, String(value));
      }
    }
  }

  const queryString = searchParams.toString();
  return queryString ? `${url}?${queryString}` : url;
}

function trimTrailingSlash(value: string) {
  return value.replace(/\/+$/, "");
}

async function readResponseBody(response: Response) {
  const text = await response.text();

  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

function resolveErrorMessage(status: number, body: unknown) {
  const detail = (body as ApiErrorBody | null)?.detail;

  if (typeof detail === "string") {
    return detail;
  }

  return `API request failed with status ${status}.`;
}

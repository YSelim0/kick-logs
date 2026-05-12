import type { Message, MessageSearchParams } from "@/types/api";

export type SearchFormState = {
  sender: string;
  channel: string;
  q: string;
  start: string;
  end: string;
};

export type ActiveFilter = {
  key: keyof SearchFormState;
  label: string;
  value: string;
};

export const EMPTY_SEARCH_STATE: SearchFormState = {
  sender: "",
  channel: "",
  q: "",
  start: "",
  end: ""
};

export const DEFAULT_SEARCH_RANGE_DAYS = 7;

const filterLabels: Record<keyof SearchFormState, string> = {
  sender: "Kullanıcı",
  channel: "Kanal",
  q: "Kelime",
  start: "Başlangıç",
  end: "Bitiş"
};

const DATE_TIME_LOCAL_MINUTE_PATTERN = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/;
const DATE_TIME_WITH_TIMEZONE_PATTERN = /(?:Z|[+-]\d{2}:?\d{2})$/i;

export function getDefaultSearchState(now = new Date()): SearchFormState {
  const end = new Date(now);
  end.setSeconds(0, 0);

  const start = new Date(end);
  start.setDate(start.getDate() - DEFAULT_SEARCH_RANGE_DAYS);

  return {
    sender: "",
    channel: "",
    q: "",
    start: toDateTimeLocalValue(start),
    end: toDateTimeLocalValue(end)
  };
}

export function readSearchState(params: URLSearchParams, now = new Date()): SearchFormState {
  const defaults = getDefaultSearchState(now);

  return {
    sender: params.get("sender") ?? "",
    channel: params.get("channel") ?? "",
    q: params.get("q") ?? "",
    start: normalizeDateInputValue(params.get("start") ?? defaults.start),
    end: normalizeDateInputValue(params.get("end") ?? defaults.end)
  };
}

export function searchStateToMessageParams(state: SearchFormState): MessageSearchParams {
  return {
    ...optionalParam("sender", state.sender),
    ...optionalParam("channel", state.channel),
    ...optionalParam("q", state.q),
    ...optionalDateParam("start", state.start),
    ...optionalDateParam("end", state.end)
  };
}

export function searchStateToUrlSearchParams(state: SearchFormState) {
  const params = new URLSearchParams();

  for (const key of Object.keys(filterLabels) as Array<keyof SearchFormState>) {
    const value = state[key].trim();
    if (value) {
      params.set(key, value);
    }
  }

  return params;
}

export function getActiveFilters(state: SearchFormState): ActiveFilter[] {
  return (Object.keys(filterLabels) as Array<keyof SearchFormState>)
    .map((key) => ({ key, label: filterLabels[key], value: state[key].trim() }))
    .filter((item) => item.value.length > 0);
}

export function getScopeText(state: SearchFormState) {
  const channel = state.channel.trim();
  return channel ? `#${channel}` : "Tüm kanallar";
}

export function getLastMatchTime(messages: Message[]) {
  const firstMessage = messages[0];
  return firstMessage ? formatMessageDate(firstMessage.message_created_at) : "-";
}

export function appendUniqueMessages(current: Message[], incoming: Message[]) {
  const existingIds = new Set(current.map((message) => message.id));
  return [
    ...current,
    ...incoming.filter((message) => {
      if (existingIds.has(message.id)) {
        return false;
      }
      existingIds.add(message.id);
      return true;
    })
  ];
}

export function formatMessageDate(value: string) {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
}

export function normalizeDateInputValue(value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return "";
  }

  if (DATE_TIME_WITH_TIMEZONE_PATTERN.test(trimmed)) {
    const date = new Date(trimmed);

    if (!Number.isNaN(date.getTime())) {
      return toDateTimeLocalValue(date);
    }
  }

  return trimmed.slice(0, 16);
}

function optionalParam(key: keyof MessageSearchParams, value: string) {
  const trimmed = value.trim();
  return trimmed ? { [key]: trimmed } : {};
}

function optionalDateParam(key: "start" | "end", value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return {};
  }

  const date = new Date(trimmed);

  if (Number.isNaN(date.getTime())) {
    return { [key]: trimmed };
  }

  if (key === "end" && DATE_TIME_LOCAL_MINUTE_PATTERN.test(trimmed)) {
    date.setSeconds(59, 999);
  }

  return { [key]: date.toISOString() };
}

function toDateTimeLocalValue(date: Date) {
  const year = date.getFullYear();
  const month = padDatePart(date.getMonth() + 1);
  const day = padDatePart(date.getDate());
  const hours = padDatePart(date.getHours());
  const minutes = padDatePart(date.getMinutes());

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function padDatePart(value: number) {
  return String(value).padStart(2, "0");
}

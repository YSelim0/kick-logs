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

const filterLabels: Record<keyof SearchFormState, string> = {
  sender: "Kullanıcı",
  channel: "Kanal",
  q: "Kelime",
  start: "Başlangıç",
  end: "Bitiş"
};

export function readSearchState(params: URLSearchParams): SearchFormState {
  return {
    sender: params.get("sender") ?? "",
    channel: params.get("channel") ?? "",
    q: params.get("q") ?? "",
    start: normalizeDateInputValue(params.get("start") ?? ""),
    end: normalizeDateInputValue(params.get("end") ?? "")
  };
}

export function searchStateToMessageParams(state: SearchFormState): MessageSearchParams {
  return {
    ...optionalParam("sender", state.sender),
    ...optionalParam("channel", state.channel),
    ...optionalParam("q", state.q),
    ...optionalParam("start", state.start),
    ...optionalParam("end", state.end)
  };
}

export function searchStateToUrlSearchParams(state: SearchFormState) {
  const params = new URLSearchParams();
  const messageParams = searchStateToMessageParams(state);

  for (const [key, value] of Object.entries(messageParams)) {
    if (value !== undefined) {
      params.set(key, String(value));
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
  if (!value) {
    return "";
  }

  return value.slice(0, 16);
}

function optionalParam(key: keyof MessageSearchParams, value: string) {
  const trimmed = value.trim();
  return trimmed ? { [key]: trimmed } : {};
}

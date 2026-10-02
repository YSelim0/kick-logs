"use client";

import { useTranslations } from "next-intl";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";
import { useUiFormat } from "@/i18n/use-ui-format";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
  Archive,
  CheckCircle2,
  Clock3,
  FileText,
  Loader2,
  MessageSquarePlus,
  RefreshCcw,
  Search,
  Send
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent } from "@/components/ui/dialog";
import {
  addUserRequestNote,
  archiveUserRequest,
  getUserRequest,
  listUserRequests,
  updateUserRequestStatus
} from "@/features/requests/api";
import { cn } from "@/lib/utils";
import type {
  UserRequest,
  UserRequestDetailResponse,
  UserRequestEvent,
  UserRequestListParams,
  UserRequestStatus,
  UserRequestType
} from "@/types/api";

type FilterState = {
  type: "" | UserRequestType;
  status: "" | UserRequestStatus;
  archived: "false" | "true" | "all";
  q: string;
  start: string;
  end: string;
};

const DEFAULT_FILTERS: FilterState = {
  type: "",
  status: "",
  archived: "false",
  q: "",
  start: "",
  end: ""
};

const TYPES: Array<"" | UserRequestType> = ["", "channel_request", "feedback"];

const STATUSES: Array<"" | UserRequestStatus> = [
  "",
  "new",
  "reviewing",
  "approved",
  "rejected",
  "done",
  "duplicate"
];

const ARCHIVES: FilterState["archived"][] = ["false", "true", "all"];

export function RequestAdmin() {
  const t = useTranslations("requestAdmin");
  const errors = useTranslations("common.errors");
  const [filters, setFilters] = useState<FilterState>(DEFAULT_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(DEFAULT_FILTERS);
  const [requests, setRequests] = useState<UserRequest[]>([]);
  const [detail, setDetail] = useState<UserRequestDetailResponse | null>(null);
  const [selectedRequestID, setSelectedRequestID] = useState<string | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [note, setNote] = useState("");
  const [nextStatus, setNextStatus] = useState<UserRequestStatus>("reviewing");
  const [error, setError] = useState<UiErrorKey | null>(null);
  const [detailError, setDetailError] = useState<UiErrorKey | null>(null);
  const [isLoadingList, setIsLoadingList] = useState(true);
  const [isLoadingDetail, setIsLoadingDetail] = useState(false);
  const [isSavingStatus, setIsSavingStatus] = useState(false);
  const [isAddingNote, setIsAddingNote] = useState(false);
  const [isArchiving, setIsArchiving] = useState(false);

  const queryParams = useMemo(() => buildListParams(appliedFilters), [appliedFilters]);

  const loadRequests = useCallback(async () => {
    setIsLoadingList(true);
    setError(null);

    try {
      const response = await listUserRequests(queryParams);
      setRequests(response.items);
    } catch (caught) {
      setError(getUiErrorKey(caught, "adminMutation"));
      setRequests([]);
    } finally {
      setIsLoadingList(false);
    }
  }, [queryParams]);

  useEffect(() => {
    void loadRequests();
  }, [loadRequests]);

  async function selectRequest(requestID: string) {
    setSelectedRequestID(requestID);
    setIsDetailModalOpen(true);
    setNote("");
    setIsLoadingDetail(true);
    setDetailError(null);

    try {
      const nextDetail = await getUserRequest(requestID);
      applyDetail(nextDetail);
    } catch (caught) {
      setDetail(null);
      setDetailError(getUiErrorKey(caught, "adminMutation"));
    } finally {
      setIsLoadingDetail(false);
    }
  }

  function submitFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAppliedFilters(filters);
    setDetail(null);
    setSelectedRequestID(null);
    setIsDetailModalOpen(false);
  }

  function resetFilters() {
    setFilters(DEFAULT_FILTERS);
    setAppliedFilters(DEFAULT_FILTERS);
    setDetail(null);
    setSelectedRequestID(null);
    setIsDetailModalOpen(false);
  }

  async function submitStatus(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail) return;

    setIsSavingStatus(true);
    setDetailError(null);

    try {
      applyDetail(
        await updateUserRequestStatus(detail.request.request_id, {
          status: nextStatus
        })
      );
    } catch (caught) {
      setDetailError(getUiErrorKey(caught, "adminMutation"));
    } finally {
      setIsSavingStatus(false);
    }
  }

  async function submitNote(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!detail || !note.trim()) return;

    setIsAddingNote(true);
    setDetailError(null);

    try {
      applyDetail(await addUserRequestNote(detail.request.request_id, { note: note.trim() }));
      setNote("");
    } catch (caught) {
      setDetailError(getUiErrorKey(caught, "adminMutation"));
    } finally {
      setIsAddingNote(false);
    }
  }

  async function submitArchive() {
    if (!detail) return;

    setIsArchiving(true);
    setDetailError(null);

    try {
      const nextDetail = await archiveUserRequest(detail.request.request_id);
      applyDetail(nextDetail);
      await loadRequests();
    } catch (caught) {
      setDetailError(getUiErrorKey(caught, "adminMutation"));
    } finally {
      setIsArchiving(false);
    }
  }

  function applyDetail(nextDetail: UserRequestDetailResponse) {
    setDetail(nextDetail);
    setSelectedRequestID(nextDetail.request.request_id);
    setNextStatus(nextDetail.request.current_status);
    setRequests((current) => mergeRequest(current, nextDetail.request));
  }

  return (
    <section className="flex flex-col gap-5">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-[22px] font-semibold text-foreground">{t("title")}</h1>
          <span className="font-mono text-[11px] text-faint">{t("description")}</span>
        </div>
        <Button
          disabled={isLoadingList}
          onClick={() => void loadRequests()}
          size="sm"
          variant="outline"
        >
          {isLoadingList ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCcw className="h-3 w-3" />
          )}
          {t("refresh")}
        </Button>
      </header>

      <form className="rounded-lg border border-border bg-panel p-4" onSubmit={submitFilters}>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <FilterSelect
            label={t("type")}
            onChange={(value) =>
              setFilters((current) => ({ ...current, type: value as FilterState["type"] }))
            }
            options={TYPES.map((value) => ({ value, label: t(`types.${value || "all"}`) }))}
            value={filters.type}
          />
          <FilterSelect
            label={t("status")}
            onChange={(value) =>
              setFilters((current) => ({ ...current, status: value as FilterState["status"] }))
            }
            options={STATUSES.map((value) => ({ value, label: t(`statuses.${value || "all"}`) }))}
            value={filters.status}
          />
          <FilterSelect
            label={t("archive")}
            onChange={(value) =>
              setFilters((current) => ({ ...current, archived: value as FilterState["archived"] }))
            }
            options={ARCHIVES.map((value) => ({ value, label: t(`archives.${value}`) }))}
            value={filters.archived}
          />
          <FilterInput
            icon={<Search className="h-3.5 w-3.5" />}
            label={t("search")}
            onChange={(value) => setFilters((current) => ({ ...current, q: value }))}
            placeholder={t("searchPlaceholder")}
            value={filters.q}
          />
          <FilterInput
            label={t("start")}
            onChange={(value) => setFilters((current) => ({ ...current, start: value }))}
            type="datetime-local"
            value={filters.start}
          />
          <FilterInput
            label={t("end")}
            onChange={(value) => setFilters((current) => ({ ...current, end: value }))}
            type="datetime-local"
            value={filters.end}
          />
          <div className="flex items-end gap-2">
            <Button className="h-[38px]" disabled={isLoadingList} type="submit">
              <Search className="h-4 w-4" />
              {t("filter")}
            </Button>
            <Button className="h-[38px]" onClick={resetFilters} type="button" variant="outline">
              {t("reset")}
            </Button>
          </div>
        </div>
      </form>

      {error ? (
        <div className="rounded-md border border-danger bg-elevated px-4 py-3 text-[13px]">
          {errors(error)}
        </div>
      ) : null}

      <section className="rounded-lg border border-border bg-panel">
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex flex-col gap-0.5">
            <h2 className="text-[14px] font-semibold text-foreground">{t("list")}</h2>
            <span className="font-mono text-[11px] text-faint">
              {isLoadingList ? t("loading") : t("count", { count: requests.length })}
            </span>
          </div>
        </div>

        <div className="hidden items-center border-b border-border px-3 py-2 md:flex">
          <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            {t("tableType")}
          </span>
          <span className="min-w-0 flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            {t("tableRequest")}
          </span>
          <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            {t("tableStatus")}
          </span>
          <span className="w-32 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            {t("tableDate")}
          </span>
        </div>

        {isLoadingList ? (
          <div className="px-4 py-12 text-center text-[13px] text-muted-foreground">
            {t("loadingList")}
          </div>
        ) : requests.length ? (
          requests.map((request) => (
            <RequestRow
              isSelected={selectedRequestID === request.request_id}
              key={request.request_id}
              onSelect={() => void selectRequest(request.request_id)}
              request={request}
            />
          ))
        ) : (
          <div className="px-4 py-12 text-center text-[13px] text-muted-foreground">
            {t("empty")}
          </div>
        )}
      </section>

      <RequestDetailModal
        detail={detail}
        detailError={detailError ? errors(detailError) : null}
        isAddingNote={isAddingNote}
        isArchiving={isArchiving}
        isLoadingDetail={isLoadingDetail}
        isOpen={isDetailModalOpen}
        isSavingStatus={isSavingStatus}
        nextStatus={nextStatus}
        note={note}
        onArchive={() => void submitArchive()}
        onNoteChange={setNote}
        onOpenChange={setIsDetailModalOpen}
        onStatusChange={setNextStatus}
        onSubmitNote={(event) => void submitNote(event)}
        onSubmitStatus={(event) => void submitStatus(event)}
      />
    </section>
  );
}

function RequestRow({
  isSelected,
  onSelect,
  request
}: {
  isSelected: boolean;
  onSelect: () => void;
  request: UserRequest;
}) {
  const f = useUiFormat();
  return (
    <button
      className={cn(
        "block w-full border-b border-border px-3 py-3 text-left transition-colors last:border-b-0 hover:bg-elevated/70",
        isSelected && "bg-elevated"
      )}
      onClick={onSelect}
      type="button"
    >
      <div className="hidden items-center gap-3 md:flex">
        <div className="w-28">
          <TypeBadge type={request.type} />
        </div>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[13px] font-medium text-foreground">{request.title}</div>
          <div className="mt-0.5 truncate font-mono text-[11px] text-faint">
            {request.channel_slug ? `#${request.channel_slug} · ` : ""}
            {request.contact ? request.contact : request.message}
          </div>
        </div>
        <div className="w-28">
          <StatusBadge status={request.current_status} archived={request.is_archived} />
        </div>
        <div className="w-32 font-mono text-[11px] text-faint">
          {f.dateTime(request.created_at)}
        </div>
      </div>

      <div className="flex flex-col gap-2 md:hidden">
        <div className="flex items-start justify-between gap-3">
          <TypeBadge type={request.type} />
          <StatusBadge status={request.current_status} archived={request.is_archived} />
        </div>
        <div>
          <div className="text-[13px] font-medium text-foreground">{request.title}</div>
          <div className="mt-0.5 font-mono text-[11px] text-faint">
            {request.channel_slug ? `#${request.channel_slug} · ` : ""}
            {f.dateTime(request.created_at)}
          </div>
        </div>
      </div>
    </button>
  );
}

function RequestDetailModal({
  detail,
  detailError,
  isAddingNote,
  isArchiving,
  isLoadingDetail,
  isOpen,
  isSavingStatus,
  nextStatus,
  note,
  onArchive,
  onNoteChange,
  onOpenChange,
  onStatusChange,
  onSubmitNote,
  onSubmitStatus
}: {
  detail: UserRequestDetailResponse | null;
  detailError: string | null;
  isAddingNote: boolean;
  isArchiving: boolean;
  isLoadingDetail: boolean;
  isOpen: boolean;
  isSavingStatus: boolean;
  nextStatus: UserRequestStatus;
  note: string;
  onArchive: () => void;
  onNoteChange: (value: string) => void;
  onOpenChange: (open: boolean) => void;
  onStatusChange: (value: UserRequestStatus) => void;
  onSubmitNote: (event: FormEvent<HTMLFormElement>) => void;
  onSubmitStatus: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const t = useTranslations("requestAdmin");
  const f = useUiFormat();
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-3xl overflow-y-auto border-border bg-panel p-0 text-foreground shadow-none">
        <DialogClose onClose={() => onOpenChange(false)} />
        <div className="border-b border-border px-5 py-4 pr-12">
          <h2 className="text-[16px] font-semibold text-foreground">{t("detail")}</h2>
          <span className="font-mono text-[11px] text-faint">
            {detail ? detail.request.request_id : t("loadingDetailShort")}
          </span>
        </div>

        {isLoadingDetail ? (
          <div className="px-5 py-12 text-center text-[13px] text-muted-foreground">
            {t("loadingDetail")}
          </div>
        ) : detail ? (
          <div className="flex flex-col gap-5 p-5">
            {detailError ? (
              <div className="rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
                {detailError}
              </div>
            ) : null}

            <div className="flex flex-col gap-3">
              <div className="flex flex-wrap items-center gap-2">
                <TypeBadge type={detail.request.type} />
                <StatusBadge
                  archived={detail.request.is_archived}
                  status={detail.request.current_status}
                />
              </div>
              <div>
                <h3 className="text-[16px] font-semibold text-foreground">
                  {detail.request.title}
                </h3>
                <p className="mt-2 whitespace-pre-wrap text-[13px] leading-relaxed text-muted-foreground">
                  {detail.request.message}
                </p>
              </div>
              <div className="grid gap-2 rounded-md border border-border bg-elevated p-3 font-mono text-[11px] text-muted-foreground sm:grid-cols-2">
                {detail.request.channel_slug ? (
                  <span>{t("channelDetail", { value: detail.request.channel_slug })}</span>
                ) : null}
                {detail.request.contact ? (
                  <span>{t("contactDetail", { value: detail.request.contact })}</span>
                ) : null}
                <span>{t("createdDetail", { value: f.dateTime(detail.request.created_at) })}</span>
                <span>
                  {t("latestDetail", { value: f.dateTime(detail.request.latest_event_at) })}
                </span>
              </div>
            </div>

            <form
              className="rounded-md border border-border bg-elevated p-3"
              onSubmit={onSubmitStatus}
            >
              <div className="mb-3 flex items-center gap-2">
                <CheckCircle2 className="h-3.5 w-3.5 text-accent" />
                <span className="text-[13px] font-semibold text-foreground">{t("status")}</span>
              </div>
              <div className="flex flex-col gap-2 sm:flex-row">
                <select
                  aria-label={t("status")}
                  className="h-[38px] flex-1 rounded-md border border-border-strong bg-panel px-3 text-[13px] text-foreground outline-none focus:border-accent"
                  onChange={(event) => onStatusChange(event.target.value as UserRequestStatus)}
                  value={nextStatus}
                >
                  {STATUSES.filter((value) => value !== "").map((value) => (
                    <option key={value} value={value}>
                      {t(`statuses.${value}`)}
                    </option>
                  ))}
                </select>
                <Button disabled={isSavingStatus} type="submit">
                  {isSavingStatus ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                  {t("save")}
                </Button>
              </div>
            </form>

            <form
              className="rounded-md border border-border bg-elevated p-3"
              onSubmit={onSubmitNote}
            >
              <div className="mb-3 flex items-center gap-2">
                <MessageSquarePlus className="h-3.5 w-3.5 text-accent" />
                <span className="text-[13px] font-semibold text-foreground">{t("note")}</span>
              </div>
              <textarea
                className="min-h-[96px] w-full resize-y rounded-md border border-border-strong bg-panel px-3 py-2.5 text-[13px] text-foreground outline-none placeholder:text-faint focus:border-accent"
                maxLength={1000}
                onChange={(event) => onNoteChange(event.target.value)}
                placeholder={t("notePlaceholder")}
                value={note}
              />
              <div className="mt-2 flex justify-end">
                <Button disabled={isAddingNote || !note.trim()} type="submit" variant="outline">
                  {isAddingNote ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                  {t("addNote")}
                </Button>
              </div>
            </form>

            <div className="rounded-md border border-border bg-elevated p-3">
              <div className="mb-3 flex items-center gap-2">
                <Clock3 className="h-3.5 w-3.5 text-accent" />
                <span className="text-[13px] font-semibold text-foreground">{t("timeline")}</span>
              </div>
              <Timeline events={detail.events} />
            </div>

            <Button
              className="border-danger text-danger hover:bg-danger/10"
              disabled={detail.request.is_archived || isArchiving}
              onClick={onArchive}
              type="button"
              variant="outline"
            >
              {isArchiving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Archive className="h-4 w-4" />
              )}
              {t("archiveAction")}
            </Button>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-2 px-5 py-12 text-center">
            <FileText className="h-6 w-6 text-faint" />
            <p className="text-[13px] text-muted-foreground">{detailError ?? t("selectDetail")}</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Timeline({ events }: { events: UserRequestEvent[] }) {
  const t = useTranslations("requestAdmin");
  const f = useUiFormat();
  if (!events.length) {
    return <p className="text-[13px] text-muted-foreground">{t("emptyTimeline")}</p>;
  }

  return (
    <ol className="flex flex-col gap-3">
      {events.map((event) => (
        <li className="border-l border-border pl-3" key={event.event_id}>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-[12px] font-medium text-foreground">
              {event.event_type === "status_changed"
                ? t("statusEvent", { status: event.status ? t(`statuses.${event.status}`) : "-" })
                : event.event_type === "note_added"
                  ? t("noteEvent")
                  : event.event_type === "archived"
                    ? t("archiveEvent")
                    : event.event_type}
            </span>
            <span className="font-mono text-[10px] text-faint">{f.dateTime(event.created_at)}</span>
          </div>
          {event.note ? (
            <p className="mt-1 whitespace-pre-wrap text-[12px] leading-relaxed text-muted-foreground">
              {event.note}
            </p>
          ) : null}
        </li>
      ))}
    </ol>
  );
}

function FilterSelect({
  label,
  onChange,
  options,
  value
}: {
  label: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  value: string;
}) {
  const id = `request-filter-${label.toLowerCase()}`;
  return (
    <div className="flex flex-col gap-1.5">
      <label className="font-mono text-[11px] text-muted-foreground" htmlFor={id}>
        {label}
      </label>
      <select
        className="h-[38px] rounded-md border border-border-strong bg-elevated px-3 text-[13px] text-foreground outline-none focus:border-accent"
        id={id}
        onChange={(event) => onChange(event.target.value)}
        value={value}
      >
        {options.map((option) => (
          <option key={option.value || "all"} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

function FilterInput({
  icon,
  label,
  onChange,
  placeholder,
  type = "text",
  value
}: {
  icon?: React.ReactNode;
  label: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: string;
  value: string;
}) {
  const id = `request-filter-${label.toLowerCase()}`;
  return (
    <div className="flex flex-col gap-1.5">
      <label className="font-mono text-[11px] text-muted-foreground" htmlFor={id}>
        {label}
      </label>
      <div className="flex h-[38px] items-center gap-2 rounded-md border border-border-strong bg-elevated px-3 focus-within:border-accent">
        {icon ? <span className="text-faint">{icon}</span> : null}
        <input
          className="min-w-0 flex-1 bg-transparent text-[13px] text-foreground outline-none placeholder:text-faint"
          id={id}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          type={type}
          value={value}
        />
      </div>
    </div>
  );
}

function TypeBadge({ type }: { type: UserRequestType }) {
  const t = useTranslations("requestAdmin");
  return (
    <span className="inline-flex h-6 items-center rounded-full border border-border bg-elevated px-2.5 font-mono text-[10px] uppercase text-muted-foreground">
      {t(`badges.${type}`)}
    </span>
  );
}

function StatusBadge({ archived, status }: { archived: boolean; status: UserRequestStatus }) {
  const t = useTranslations("requestAdmin");
  return (
    <span
      className={cn(
        "inline-flex h-6 items-center gap-1.5 rounded-full border px-2.5 font-mono text-[10px] uppercase",
        archived
          ? "border-border bg-elevated text-faint"
          : status === "new"
            ? "border-accent bg-accent/10 text-accent"
            : "border-border bg-elevated text-muted-foreground"
      )}
    >
      <span
        aria-hidden
        className={cn("h-1.5 w-1.5 rounded-full", archived ? "bg-faint" : "bg-accent")}
      />
      {archived ? t("archive") : t(`statuses.${status}`)}
    </span>
  );
}

function buildListParams(filters: FilterState): UserRequestListParams {
  return {
    type: filters.type || undefined,
    status: filters.status || undefined,
    archived: filters.archived === "all" ? undefined : filters.archived === "true",
    q: optionalValue(filters.q),
    start: localDateTimeToISO(filters.start),
    end: localDateTimeToISO(filters.end),
    limit: 50
  };
}

function localDateTimeToISO(value: string) {
  if (!value) return undefined;
  return new Date(value).toISOString();
}

function optionalValue(value: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function mergeRequest(current: UserRequest[], request: UserRequest) {
  const next = current.map((item) => (item.request_id === request.request_id ? request : item));
  return next.some((item) => item.request_id === request.request_id) ? next : [request, ...next];
}

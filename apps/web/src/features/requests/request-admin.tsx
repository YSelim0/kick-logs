"use client";

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

const TYPE_OPTIONS: Array<{ value: "" | UserRequestType; label: string }> = [
  { value: "", label: "Tüm tipler" },
  { value: "channel_request", label: "Kanal Talebi" },
  { value: "feedback", label: "Geri Bildirim" }
];

const STATUS_OPTIONS: Array<{ value: "" | UserRequestStatus; label: string }> = [
  { value: "", label: "Tüm durumlar" },
  { value: "new", label: "Yeni" },
  { value: "reviewing", label: "İncelemede" },
  { value: "approved", label: "Onaylandı" },
  { value: "rejected", label: "Reddedildi" },
  { value: "done", label: "Tamamlandı" },
  { value: "duplicate", label: "Tekrar" }
];

const ARCHIVE_OPTIONS: Array<{ value: FilterState["archived"]; label: string }> = [
  { value: "false", label: "Aktif" },
  { value: "true", label: "Arşiv" },
  { value: "all", label: "Tümü" }
];

export function RequestAdmin() {
  const [filters, setFilters] = useState<FilterState>(DEFAULT_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(DEFAULT_FILTERS);
  const [requests, setRequests] = useState<UserRequest[]>([]);
  const [detail, setDetail] = useState<UserRequestDetailResponse | null>(null);
  const [selectedRequestID, setSelectedRequestID] = useState<string | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [note, setNote] = useState("");
  const [nextStatus, setNextStatus] = useState<UserRequestStatus>("reviewing");
  const [error, setError] = useState<string | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
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
      setError(resolveRequestAdminError(caught));
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
      setDetailError(resolveRequestAdminError(caught));
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
      setDetailError(resolveRequestAdminError(caught));
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
      setDetailError(resolveRequestAdminError(caught));
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
      setDetailError(resolveRequestAdminError(caught));
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
          <h1 className="text-[22px] font-semibold text-foreground">Talepler</h1>
          <span className="font-mono text-[11px] text-faint">
            Kanal talepleri ve geri bildirim akışı
          </span>
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
          Yenile
        </Button>
      </header>

      <form className="rounded-lg border border-border bg-panel p-4" onSubmit={submitFilters}>
        <div className="grid gap-3 lg:grid-cols-[160px_160px_140px_minmax(0,1fr)_170px_170px_auto]">
          <FilterSelect
            label="Tip"
            onChange={(value) =>
              setFilters((current) => ({ ...current, type: value as FilterState["type"] }))
            }
            options={TYPE_OPTIONS}
            value={filters.type}
          />
          <FilterSelect
            label="Durum"
            onChange={(value) =>
              setFilters((current) => ({ ...current, status: value as FilterState["status"] }))
            }
            options={STATUS_OPTIONS}
            value={filters.status}
          />
          <FilterSelect
            label="Arşiv"
            onChange={(value) =>
              setFilters((current) => ({ ...current, archived: value as FilterState["archived"] }))
            }
            options={ARCHIVE_OPTIONS}
            value={filters.archived}
          />
          <FilterInput
            icon={<Search className="h-3.5 w-3.5" />}
            label="Arama"
            onChange={(value) => setFilters((current) => ({ ...current, q: value }))}
            placeholder="başlık, mesaj, kanal, iletişim"
            value={filters.q}
          />
          <FilterInput
            label="Başlangıç"
            onChange={(value) => setFilters((current) => ({ ...current, start: value }))}
            type="datetime-local"
            value={filters.start}
          />
          <FilterInput
            label="Bitiş"
            onChange={(value) => setFilters((current) => ({ ...current, end: value }))}
            type="datetime-local"
            value={filters.end}
          />
          <div className="flex items-end gap-2">
            <Button className="h-[38px]" disabled={isLoadingList} type="submit">
              <Search className="h-4 w-4" />
              Filtrele
            </Button>
            <Button className="h-[38px]" onClick={resetFilters} type="button" variant="outline">
              Sıfırla
            </Button>
          </div>
        </div>
      </form>

      {error ? (
        <div className="rounded-md border border-danger bg-elevated px-4 py-3 text-[13px]">
          {error}
        </div>
      ) : null}

      <section className="rounded-lg border border-border bg-panel">
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex flex-col gap-0.5">
            <h2 className="text-[14px] font-semibold text-foreground">Talep Listesi</h2>
            <span className="font-mono text-[11px] text-faint">
              {isLoadingList ? "yükleniyor" : `${requests.length} kayıt`}
            </span>
          </div>
        </div>

        <div className="hidden items-center border-b border-border px-3 py-2 md:flex">
          <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            TİP
          </span>
          <span className="min-w-0 flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            TALEP
          </span>
          <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            DURUM
          </span>
          <span className="w-32 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            TARİH
          </span>
        </div>

        {isLoadingList ? (
          <div className="px-4 py-12 text-center text-[13px] text-muted-foreground">
            Talepler yükleniyor...
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
            Bu filtrelerle talep bulunamadı.
          </div>
        )}
      </section>

      <RequestDetailModal
        detail={detail}
        detailError={detailError}
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
          {formatDate(request.created_at)}
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
            {formatDate(request.created_at)}
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
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-3xl overflow-y-auto border-border bg-panel p-0 text-foreground shadow-none">
        <DialogClose onClose={() => onOpenChange(false)} />
        <div className="border-b border-border px-5 py-4 pr-12">
          <h2 className="text-[16px] font-semibold text-foreground">Talep Detayı</h2>
          <span className="font-mono text-[11px] text-faint">
            {detail ? detail.request.request_id : "detay yükleniyor"}
          </span>
        </div>

        {isLoadingDetail ? (
          <div className="px-5 py-12 text-center text-[13px] text-muted-foreground">
            Detay yükleniyor...
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
                  <span>Kanal: #{detail.request.channel_slug}</span>
                ) : null}
                {detail.request.contact ? <span>İletişim: {detail.request.contact}</span> : null}
                <span>Oluşturulma: {formatDate(detail.request.created_at)}</span>
                <span>Son hareket: {formatDate(detail.request.latest_event_at)}</span>
              </div>
            </div>

            <form
              className="rounded-md border border-border bg-elevated p-3"
              onSubmit={onSubmitStatus}
            >
              <div className="mb-3 flex items-center gap-2">
                <CheckCircle2 className="h-3.5 w-3.5 text-accent" />
                <span className="text-[13px] font-semibold text-foreground">Durum</span>
              </div>
              <div className="flex flex-col gap-2 sm:flex-row">
                <select
                  className="h-[38px] flex-1 rounded-md border border-border-strong bg-panel px-3 text-[13px] text-foreground outline-none focus:border-accent"
                  onChange={(event) => onStatusChange(event.target.value as UserRequestStatus)}
                  value={nextStatus}
                >
                  {STATUS_OPTIONS.filter((option) => option.value !== "").map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <Button disabled={isSavingStatus} type="submit">
                  {isSavingStatus ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                  Kaydet
                </Button>
              </div>
            </form>

            <form
              className="rounded-md border border-border bg-elevated p-3"
              onSubmit={onSubmitNote}
            >
              <div className="mb-3 flex items-center gap-2">
                <MessageSquarePlus className="h-3.5 w-3.5 text-accent" />
                <span className="text-[13px] font-semibold text-foreground">Not</span>
              </div>
              <textarea
                className="min-h-[96px] w-full resize-y rounded-md border border-border-strong bg-panel px-3 py-2.5 text-[13px] text-foreground outline-none placeholder:text-faint focus:border-accent"
                maxLength={1000}
                onChange={(event) => onNoteChange(event.target.value)}
                placeholder="İnceleme notu ekle"
                value={note}
              />
              <div className="mt-2 flex justify-end">
                <Button disabled={isAddingNote || !note.trim()} type="submit" variant="outline">
                  {isAddingNote ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                  Not ekle
                </Button>
              </div>
            </form>

            <div className="rounded-md border border-border bg-elevated p-3">
              <div className="mb-3 flex items-center gap-2">
                <Clock3 className="h-3.5 w-3.5 text-accent" />
                <span className="text-[13px] font-semibold text-foreground">Timeline</span>
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
              Arşivle
            </Button>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-2 px-5 py-12 text-center">
            <FileText className="h-6 w-6 text-faint" />
            <p className="text-[13px] text-muted-foreground">
              {detailError ?? "Detay görüntülemek için listeden bir talep seç."}
            </p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Timeline({ events }: { events: UserRequestEvent[] }) {
  if (!events.length) {
    return <p className="text-[13px] text-muted-foreground">Henüz admin hareketi yok.</p>;
  }

  return (
    <ol className="flex flex-col gap-3">
      {events.map((event) => (
        <li className="border-l border-border pl-3" key={event.event_id}>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-[12px] font-medium text-foreground">{eventLabel(event)}</span>
            <span className="font-mono text-[10px] text-faint">{formatDate(event.created_at)}</span>
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
  return (
    <span className="inline-flex h-6 items-center rounded-full border border-border bg-elevated px-2.5 font-mono text-[10px] uppercase text-muted-foreground">
      {typeLabel(type)}
    </span>
  );
}

function StatusBadge({ archived, status }: { archived: boolean; status: UserRequestStatus }) {
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
      {archived ? "Arşiv" : statusLabel(status)}
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

function typeLabel(type: UserRequestType) {
  return type === "channel_request" ? "Kanal" : "Feedback";
}

function statusLabel(status: UserRequestStatus) {
  const match = STATUS_OPTIONS.find((option) => option.value === status);
  return match?.label ?? status;
}

function eventLabel(event: UserRequestEvent) {
  if (event.event_type === "status_changed") {
    return `Durum: ${event.status ? statusLabel(event.status) : "-"}`;
  }
  if (event.event_type === "note_added") {
    return "Not eklendi";
  }
  if (event.event_type === "archived") {
    return "Arşivlendi";
  }
  return event.event_type;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

function resolveRequestAdminError(error: unknown) {
  if (error instanceof Error) return error.message;
  return "Talep işlemi tamamlanamadı.";
}

"use client";

/* eslint-disable @next/next/no-img-element */

import { ChevronDown, Search } from "lucide-react";
import { useState } from "react";

import { MessageContent } from "@/features/search/message-content";
import { getReplyContext } from "@/features/search/reply-metadata";
import { formatMessageDate } from "@/features/search/search-params";
import { cn } from "@/lib/utils";
import type { Message } from "@/types/api";

type MessageListProps = {
  messages: Message[];
  isInitialLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  hasSearched: boolean;
  error: string | null;
  sentinelRef: React.RefObject<HTMLDivElement>;
  onRetry: () => void;
};

export function MessageList({
  messages,
  isInitialLoading,
  isLoadingMore,
  hasMore,
  hasSearched,
  error,
  sentinelRef,
  onRetry
}: MessageListProps) {
  return (
    <section className="rounded-lg border border-border bg-black p-4">
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3 border-b border-border pb-3">
        <div>
          <div className="flex items-center gap-2 text-base font-semibold">
            <span className="h-2 w-2 rounded-full bg-primary" />
            Sonuçlar
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {hasSearched
              ? `${messages.length} mesaj - en yeni kayıtlar önce - kaydırdıkça eski mesajlar yüklenir`
              : "Arama yaptıktan sonra sonuçlar burada listelenir"}
          </p>
        </div>
      </div>

      <div className="overflow-hidden rounded-md border border-border">
        {!hasSearched && !isInitialLoading && !error ? (
          <div className="flex min-h-[180px] flex-col items-center justify-center gap-3 bg-kick-background px-4 py-12 text-center">
            <div className="flex h-11 w-11 items-center justify-center rounded-md border border-border bg-black text-primary">
              <Search className="h-5 w-5" />
            </div>
            <p className="text-sm font-medium text-foreground">
              Arama yapmak için yukarıdaki formu kullanın.
            </p>
          </div>
        ) : (
          <>
            <div className="hidden grid-cols-[46px_142px_130px_minmax(0,1fr)_154px] bg-kick-background px-3 py-2 text-xs font-medium text-muted-foreground md:grid">
              <div />
              <div>Gönderen</div>
              <div>Kanal</div>
              <div>Mesaj</div>
              <div>Tarih</div>
            </div>

            {messages.map((message, index) => (
              <MessageRow isAlt={index % 2 === 1} key={message.id} message={message} />
            ))}

            {!isInitialLoading && !error && messages.length === 0 ? (
              <div className="px-4 py-10 text-center text-sm text-muted-foreground">
                Bu filtrelerle mesaj bulunamadı.
              </div>
            ) : null}
          </>
        )}

        {isInitialLoading ? (
          <div className="px-4 py-10 text-center text-sm text-muted-foreground">
            Mesajlar yükleniyor...
          </div>
        ) : null}

        {error ? (
          <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-4 text-sm text-muted-foreground">
            <span>{error}</span>
            <button
              className="rounded-md border border-border px-3 py-2 text-xs text-primary hover:border-primary"
              onClick={onRetry}
              type="button"
            >
              Tekrar dene
            </button>
          </div>
        ) : null}
      </div>

      <div className="mt-3 flex justify-center" ref={sentinelRef}>
        {isLoadingMore ? (
          <div className="inline-flex h-8 items-center gap-2 rounded-md bg-kick-background px-3 text-xs text-muted-foreground">
            <ChevronDown className="h-4 w-4 text-primary" />
            Daha eski mesajlar yükleniyor
          </div>
        ) : null}

        {!isLoadingMore && !hasMore && messages.length > 0 ? (
          <div className="inline-flex h-8 items-center rounded-md bg-kick-background px-3 text-xs text-muted-foreground">
            Sonuçların sonuna ulaşıldı
          </div>
        ) : null}
      </div>
    </section>
  );
}

function MessageRow({ message, isAlt }: { message: Message; isAlt: boolean }) {
  const replyContext = getReplyContext(message);
  const replyTitle = replyContext
    ? `@${replyContext.senderUsername}: ${replyContext.content}`
    : undefined;

  return (
    <div
      className={cn(
        "grid min-h-[54px] grid-cols-[40px_minmax(0,1fr)] gap-3 border-t border-border/70 px-3 py-3 text-sm md:grid-cols-[46px_142px_130px_minmax(0,1fr)_154px] md:items-center md:gap-0 md:py-2",
        isAlt ? "bg-kick-background" : "bg-black"
      )}
    >
      <div className="flex items-start md:items-center">
        <SenderAvatar message={message} />
      </div>

      <div className="min-w-0 md:pr-3">
        <div className="truncate font-medium text-foreground">
          {message.sender.username || message.sender_username_snapshot}
        </div>
        <div className="truncate text-xs text-muted-foreground md:hidden">
          #{message.channel.slug} - {formatMessageDate(message.message_created_at)}
        </div>
      </div>

      <div className="hidden min-w-0 pr-3 text-accent md:block">
        <span className="truncate">#{message.channel.slug}</span>
      </div>

      <div className="col-span-2 min-w-0 text-foreground md:col-span-1 md:pr-4">
        {replyContext ? (
          <div
            className="mb-1 min-w-0 truncate text-xs leading-5 text-muted-foreground/70"
            title={replyTitle}
          >
            <span className="font-medium text-muted-foreground/80">
              @{replyContext.senderUsername}:
            </span>{" "}
            {replyContext.content}
          </div>
        ) : null}
        <MessageContent content={message.content} emotes={message.emotes} />
      </div>

      <div className="hidden text-xs text-muted-foreground md:block">
        {formatMessageDate(message.message_created_at)}
      </div>
    </div>
  );
}

function SenderAvatar({ message }: { message: Message }) {
  const [failed, setFailed] = useState(false);
  const imageUrl = message.sender.profile_image_url;
  const initial = (message.sender.username || message.sender_username_snapshot || "?")
    .slice(0, 1)
    .toUpperCase();

  if (imageUrl && !failed) {
    return (
      <img
        alt={`${message.sender.username} profil`}
        className="h-8 w-8 rounded-full border border-border object-cover"
        height={32}
        loading="lazy"
        onError={() => setFailed(true)}
        src={imageUrl}
        width={32}
      />
    );
  }

  return (
    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-semibold text-secondary-foreground">
      {initial}
    </div>
  );
}

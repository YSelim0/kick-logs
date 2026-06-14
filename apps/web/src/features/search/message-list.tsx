"use client";

import { Search } from "lucide-react";
import Link from "next/link";
import type { CSSProperties } from "react";

import { MessageContent } from "@/features/search/message-content";
import { getReplyContext } from "@/features/search/reply-metadata";
import { formatMessageDate } from "@/features/search/search-params";
import { buildUserProfileHref } from "@/lib/kick-profile-slugs";
import { buildChannelProfileHref } from "@/lib/channel-profile-slugs";
import type { Message } from "@/types/api";

type MessageListProps = {
  messages: Message[];
  isInitialLoading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  hasSearched: boolean;
  highlightQuery?: string;
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
  highlightQuery = "",
  error,
  sentinelRef,
  onRetry
}: MessageListProps) {
  const emptyShell = !hasSearched && !isInitialLoading && !error;

  return (
    <div className="flex flex-col gap-3">
      <div className="overflow-hidden rounded-lg border border-border bg-panel">
        {emptyShell ? (
          <div className="flex min-h-[180px] flex-col items-center justify-center gap-3 px-4 py-12 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-md border border-border bg-elevated text-muted-foreground">
              <Search className="h-4 w-4" />
            </div>
            <p className="text-[13px] text-muted-foreground">
              Arama yapmak için yukarıdaki formu kullanın.
            </p>
          </div>
        ) : (
          <>
            {messages.map((message) => (
              <MessageRow highlightQuery={highlightQuery} key={message.id} message={message} />
            ))}

            {!isInitialLoading && !error && messages.length === 0 ? (
              <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
                Bu filtrelerle mesaj bulunamadı.
              </div>
            ) : null}
          </>
        )}

        {isInitialLoading ? (
          <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
            Mesajlar yükleniyor…
          </div>
        ) : null}

        {error ? (
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-4 py-4 text-[13px] text-muted-foreground">
            <span>{error}</span>
            <button
              className="rounded-md border border-border-strong px-3 py-1.5 text-xs text-foreground hover:bg-elevated"
              onClick={onRetry}
              type="button"
            >
              Tekrar dene
            </button>
          </div>
        ) : null}
      </div>

      <div className="flex justify-center" ref={sentinelRef}>
        {isLoadingMore ? (
          <div className="inline-flex items-center gap-2 px-3 py-2 font-mono text-[11px] text-muted-foreground">
            <span className="h-2 w-2 rounded-full bg-accent" />
            daha eski mesajlar yükleniyor…
          </div>
        ) : null}

        {!isLoadingMore && !hasMore && messages.length > 0 ? (
          <div className="inline-flex items-center px-3 py-2 font-mono text-[11px] text-muted-foreground">
            sonuçların sonuna ulaşıldı
          </div>
        ) : null}
      </div>
    </div>
  );
}

function MessageRow({ message, highlightQuery }: { message: Message; highlightQuery: string }) {
  const replyContext = getReplyContext(message);
  const replyTitle = replyContext
    ? `@${replyContext.senderUsername}: ${replyContext.content}`
    : undefined;
  const replySenderProfileHref = buildUserProfileHref(replyContext?.senderSlug);
  const senderProfileHref = buildUserProfileHref(message.sender.slug);
  const channelProfileHref = buildChannelProfileHref(message.channel.slug);
  const senderName = message.sender.username || message.sender_username_snapshot;
  const senderNameStyle = senderColorStyle(message.sender_color_snapshot);
  const channelLabel = `#${message.channel.slug}`;

  return (
    <div className="grid grid-cols-1 gap-2 border-b border-border px-4 py-3 text-[13px] last:border-b-0 md:grid-cols-[140px_minmax(0,1fr)_auto] md:items-start md:gap-4">
      <div className="flex min-w-0 flex-col gap-0.5">
        {senderProfileHref ? (
          <Link
            className="truncate font-medium text-foreground hover:underline"
            href={senderProfileHref}
            style={senderNameStyle}
          >
            {senderName}
          </Link>
        ) : (
          <span className="truncate font-medium text-foreground" style={senderNameStyle}>
            {senderName}
          </span>
        )}
        {channelProfileHref ? (
          <Link
            className="truncate font-mono text-[11px] text-accent hover:underline"
            href={channelProfileHref}
          >
            {channelLabel}
          </Link>
        ) : (
          <span className="truncate font-mono text-[11px] text-accent">{channelLabel}</span>
        )}
      </div>

      <div className="min-w-0 text-foreground">
        {replyContext ? (
          <div
            className="mb-1 flex w-fit max-w-full items-center gap-1.5 rounded-md bg-elevated px-2 py-1 text-[12px] text-muted-foreground"
            title={replyTitle}
          >
            <span className="text-faint">↳</span>
            {replySenderProfileHref ? (
              <Link
                className="font-medium text-foreground/80 hover:underline"
                href={replySenderProfileHref}
              >
                @{replyContext.senderUsername}:
              </Link>
            ) : (
              <span className="font-medium text-foreground/80">
                @{replyContext.senderUsername}:
              </span>
            )}
            <MessageContent
              className="min-w-0 flex-1 flex-nowrap overflow-hidden gap-x-1 leading-5"
              content={replyContext.content}
              emoteClassName="h-4 w-4"
              emotes={[]}
              textPartClassName="min-w-0 truncate"
            />
          </div>
        ) : null}
        <MessageContent
          content={message.content}
          emotes={message.emotes}
          highlight={highlightQuery}
        />
      </div>

      <div className="text-right font-mono text-[11px] text-faint md:whitespace-nowrap">
        {formatMessageDate(message.message_created_at)}
      </div>
    </div>
  );
}

function senderColorStyle(color: string | null): CSSProperties | undefined {
  if (!color || !/^#[0-9a-fA-F]{3,8}$/.test(color)) {
    return undefined;
  }

  return { color };
}

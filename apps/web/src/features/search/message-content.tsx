"use client";

/* eslint-disable @next/next/no-img-element */

import { useMemo, useState } from "react";

import type { MessageEmote } from "@/types/api";

type TextPart = {
  type: "text";
  value: string;
};

type EmotePart = {
  type: "emote";
  id: string;
  name: string;
  token: string;
  imageUrl: string;
};

export type MessageContentPart = TextPart | EmotePart;

type MessageContentProps = {
  content: string;
  emotes: MessageEmote[];
};

const emoteTokenPattern = /\[emote:(\d+):([^\]]+)\]/g;

export function MessageContent({ content, emotes }: MessageContentProps) {
  const [failedTokens, setFailedTokens] = useState<Set<string>>(() => new Set());
  const parts = useMemo(() => splitMessageContent(content, emotes), [content, emotes]);

  return (
    <span className="inline-flex min-w-0 flex-wrap items-center gap-x-1 gap-y-0.5 leading-6">
      {parts.map((part, index) => {
        if (part.type === "text") {
          return <span key={`${part.value}-${index}`}>{part.value}</span>;
        }

        if (failedTokens.has(part.token)) {
          return (
            <span
              className="inline-flex h-5 items-center rounded-sm bg-primary px-1.5 text-[11px] font-semibold text-primary-foreground"
              key={`${part.token}-${index}`}
            >
              {part.name}
            </span>
          );
        }

        return (
          <img
            alt={part.name}
            className="inline-block h-6 w-6 shrink-0 object-contain align-middle"
            height={24}
            key={`${part.token}-${index}`}
            loading="lazy"
            onError={() => {
              setFailedTokens((current) => new Set(current).add(part.token));
            }}
            src={part.imageUrl}
            width={24}
          />
        );
      })}
    </span>
  );
}

export function splitMessageContent(
  content: string,
  emotes: MessageEmote[]
): MessageContentPart[] {
  const parts: MessageContentPart[] = [];
  const emotesByToken = new Map(emotes.map((emote) => [emote.token, emote]));
  let lastIndex = 0;

  for (const match of content.matchAll(emoteTokenPattern)) {
    const token = match[0];
    const matchIndex = match.index ?? 0;

    if (matchIndex > lastIndex) {
      parts.push({ type: "text", value: content.slice(lastIndex, matchIndex) });
    }

    const id = match[1];
    const name = match[2];
    const emote = emotesByToken.get(token);

    parts.push({
      type: "emote",
      id,
      name: emote?.name ?? name,
      token,
      imageUrl: emote?.image_url ?? `https://files.kick.com/emotes/${id}/fullsize`
    });

    lastIndex = matchIndex + token.length;
  }

  if (lastIndex < content.length) {
    parts.push({ type: "text", value: content.slice(lastIndex) });
  }

  return parts.length ? parts : [{ type: "text", value: content }];
}

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

export type MessageTextSegment =
  | {
      type: "text";
      value: string;
    }
  | {
      type: "link";
      value: string;
      href: string;
    };

type MessageContentProps = {
  content: string;
  emotes: MessageEmote[];
  highlight?: string;
};

const emoteTokenPattern = /\[emote:(\d+):([^\]]+)\]/g;
const urlPattern = /https?:\/\/[^\s<>"']+/gi;
const trailingUrlPunctuationPattern = /[),.!?;:]+$/;

export function MessageContent({ content, emotes, highlight = "" }: MessageContentProps) {
  const [failedTokens, setFailedTokens] = useState<Set<string>>(() => new Set());
  const parts = useMemo(() => splitMessageContent(content, emotes), [content, emotes]);
  const highlightText = highlight.trim();

  return (
    <span className="inline-flex min-w-0 flex-wrap items-center gap-x-1 gap-y-0.5 leading-6">
      {parts.map((part, index) => {
        if (part.type === "text") {
          return (
            <span key={`${part.value}-${index}`}>
              {renderTextSegments(part.value, highlightText)}
            </span>
          );
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

export function splitMessageContent(content: string, emotes: MessageEmote[]): MessageContentPart[] {
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

export function splitTextContent(value: string): MessageTextSegment[] {
  const segments: MessageTextSegment[] = [];
  let lastIndex = 0;

  for (const match of value.matchAll(urlPattern)) {
    const rawUrl = match[0];
    const matchIndex = match.index ?? 0;

    if (matchIndex > lastIndex) {
      segments.push({ type: "text", value: value.slice(lastIndex, matchIndex) });
    }

    const { linkText, trailingText } = trimTrailingUrlPunctuation(rawUrl);
    const href = safeUrlHref(linkText);

    if (href) {
      segments.push({ type: "link", value: linkText, href });
    } else {
      segments.push({ type: "text", value: linkText });
    }

    if (trailingText) {
      segments.push({ type: "text", value: trailingText });
    }

    lastIndex = matchIndex + rawUrl.length;
  }

  if (lastIndex < value.length) {
    segments.push({ type: "text", value: value.slice(lastIndex) });
  }

  return segments.length ? segments : [{ type: "text", value }];
}

function renderTextSegments(value: string, highlight: string) {
  return splitTextContent(value).map((segment, index) => {
    if (segment.type === "link") {
      return (
        <a
          className="break-all text-primary underline-offset-2 hover:underline"
          href={segment.href}
          key={`${segment.href}-${index}`}
          rel="noreferrer noopener"
          target="_blank"
        >
          {renderHighlightedText(segment.value, highlight)}
        </a>
      );
    }

    return (
      <span key={`${segment.value}-${index}`}>
        {renderHighlightedText(segment.value, highlight)}
      </span>
    );
  });
}

function renderHighlightedText(value: string, highlight: string) {
  if (!highlight) {
    return value;
  }

  const query = highlight.toLocaleLowerCase("tr-TR");
  const segments = value.split(new RegExp(`(${escapeRegExp(highlight)})`, "gi"));

  return segments.map((segment, index) => {
    if (segment.toLocaleLowerCase("tr-TR") !== query) {
      return segment;
    }

    return (
      <mark
        className="rounded-sm bg-primary/20 px-0.5 font-semibold text-primary"
        key={`${segment}-${index}`}
      >
        {segment}
      </mark>
    );
  });
}

function trimTrailingUrlPunctuation(value: string) {
  const trailingMatch = value.match(trailingUrlPunctuationPattern);
  const trailingText = trailingMatch?.[0] ?? "";
  const linkText = trailingText ? value.slice(0, -trailingText.length) : value;

  return { linkText, trailingText };
}

function safeUrlHref(value: string) {
  try {
    const url = new URL(value);
    return url.protocol === "http:" || url.protocol === "https:" ? url.toString() : null;
  } catch {
    return null;
  }
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

import type { JsonRecord, Message } from "@/types/api";

export type ReplyContext = {
  senderUsername: string;
  senderSlug: string;
  messageId: string;
  content: string;
};

export function getReplyContext(message: Message): ReplyContext | null {
  if (message.message_type !== "reply") {
    return null;
  }

  const metadata = message.reply_metadata;
  const originalSender = readRecord(metadata.original_sender);
  const originalMessage = readRecord(metadata.original_message);

  if (!originalSender || !originalMessage) {
    return null;
  }

  const senderUsername = readRequiredString(originalSender.username);
  const messageId = readRequiredString(originalMessage.id);
  const content = readRequiredString(originalMessage.content);

  if (!senderUsername || !messageId || !content) {
    return null;
  }

  return {
    senderUsername,
    senderSlug: readOptionalString(originalSender.slug) ?? normalizeSenderSlug(senderUsername),
    messageId,
    content
  };
}

function readRecord(value: unknown): JsonRecord | null {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as JsonRecord;
  }

  return null;
}

function readRequiredString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function readOptionalString(value: unknown): string | null {
  const text = readRequiredString(value);
  return text ? text : null;
}

function normalizeSenderSlug(username: string): string {
  return username.trim().toLowerCase();
}

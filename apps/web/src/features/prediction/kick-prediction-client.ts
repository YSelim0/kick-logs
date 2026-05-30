import { ApiClientError } from "@/lib/api-client";
import type { Prediction, PredictionOutcome, PredictionTopUser } from "@/types/api";

const KICK_API_BASE = "https://kick.com/api/v2";
const MAX_SLUG_LENGTH = 160;

type Fetcher = typeof fetch;

type ValidKickTopUser = {
  id: number;
  username: string;
  amount: number;
};

type ValidKickOutcome = {
  id: string;
  title: string;
  total_vote_amount: number;
  vote_count: number;
  return_rate: number;
  top_users: ValidKickTopUser[];
};

type ValidKickPrediction = {
  id: string;
  channel_id: number;
  title: string;
  outcomes: ValidKickOutcome[];
  duration: number;
  created_at: string | null;
  updated_at: string | null;
  locked_at: string | null;
  state: string;
  winning_outcome_id: string | null;
};

const channelValidationCache = new WeakMap<Fetcher, Map<string, Promise<void>>>();

export async function fetchKickPrediction(
  slug: string,
  fetchImpl: Fetcher = fetch,
  signal?: AbortSignal
): Promise<Prediction> {
  const normalized = slug.trim().toLowerCase();
  if (!normalized || normalized.length > MAX_SLUG_LENGTH) {
    throw new ApiClientError(404, { detail: "Invalid channel slug." });
  }

  await ensureChannelExistsCached(normalized, fetchImpl, signal);
  return fetchLatestPrediction(normalized, fetchImpl, signal);
}

function ensureChannelExistsCached(
  slug: string,
  fetchImpl: Fetcher,
  signal?: AbortSignal
): Promise<void> {
  let cache = channelValidationCache.get(fetchImpl);
  if (!cache) {
    cache = new Map();
    channelValidationCache.set(fetchImpl, cache);
  }

  const cached = cache.get(slug);
  if (cached) {
    return cached;
  }

  const validation = ensureChannelExists(slug, fetchImpl, signal).catch((error) => {
    cache.delete(slug);
    throw error;
  });
  cache.set(slug, validation);
  return validation;
}

async function ensureChannelExists(slug: string, fetchImpl: Fetcher, signal?: AbortSignal) {
  const response = await safeFetch(
    fetchImpl,
    `${KICK_API_BASE}/channels/${encodeURIComponent(slug)}`,
    signal
  );
  if (response.status === 404) {
    throw new ApiClientError(404, { detail: "Channel not found." });
  }
  if (!response.ok) {
    throw new ApiClientError(502, { detail: "Kick prediction request was blocked." });
  }
}

async function fetchLatestPrediction(
  slug: string,
  fetchImpl: Fetcher,
  signal?: AbortSignal
): Promise<Prediction> {
  const response = await safeFetch(
    fetchImpl,
    `${KICK_API_BASE}/channels/${encodeURIComponent(slug)}/predictions/latest`,
    signal
  );
  if (response.status === 404) {
    throw new ApiClientError(404, { detail: "No active prediction found for this channel." });
  }
  if (!response.ok) {
    throw new ApiClientError(502, { detail: "Kick prediction request was blocked." });
  }

  let envelope: unknown;
  try {
    envelope = await response.json();
  } catch {
    throw new ApiClientError(502, { detail: "Kick prediction response was malformed." });
  }

  const raw = readPredictionFromEnvelope(envelope);
  if (!raw) {
    throw new ApiClientError(404, { detail: "No active prediction found for this channel." });
  }

  return normalizePrediction(raw);
}

async function safeFetch(fetchImpl: Fetcher, url: string, signal?: AbortSignal): Promise<Response> {
  try {
    return await fetchImpl(url, { headers: { accept: "application/json" }, signal });
  } catch {
    throw new ApiClientError(502, { detail: "Kick prediction request failed." });
  }
}

function normalizePrediction(raw: unknown): Prediction {
  const prediction = validatePrediction(raw);
  const winningOutcomeId = prediction.winning_outcome_id;
  const outcomes = prediction.outcomes.map((outcome) =>
    normalizeOutcome(outcome, winningOutcomeId)
  );

  const totalPoints = outcomes.reduce((sum, outcome) => sum + outcome.totalVoteAmount, 0);
  const totalVotes = outcomes.reduce((sum, outcome) => sum + outcome.voteCount, 0);

  for (const outcome of outcomes) {
    outcome.pointShare = totalPoints > 0 ? outcome.totalVoteAmount / totalPoints : 0;
  }

  return {
    id: prediction.id,
    channelId: prediction.channel_id,
    title: prediction.title,
    durationSeconds: prediction.duration,
    state: prediction.state,
    winningOutcomeId,
    createdAt: prediction.created_at,
    lockedAt: prediction.locked_at,
    updatedAt: prediction.updated_at,
    totalPoints,
    totalVotes,
    outcomes
  };
}

function normalizeOutcome(
  outcome: ValidKickOutcome,
  winningOutcomeId: string | null
): PredictionOutcome {
  const id = outcome.id;
  return {
    id,
    title: outcome.title,
    totalVoteAmount: outcome.total_vote_amount,
    voteCount: outcome.vote_count,
    returnRate: outcome.return_rate,
    pointShare: 0,
    isWinner: winningOutcomeId !== null && id === winningOutcomeId,
    topUsers: outcome.top_users.map(normalizeTopUser)
  };
}

function normalizeTopUser(user: ValidKickTopUser): PredictionTopUser {
  return {
    id: user.id,
    username: user.username,
    amount: user.amount
  };
}

function nullableString(value: string | null | undefined): string | null {
  if (value === null || value === undefined) {
    return null;
  }
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
}

function readPredictionFromEnvelope(value: unknown): unknown | null {
  const record = requireRecord(value);
  const error = readNullableString(record, "error");
  if (error) {
    throw new ApiClientError(502, { detail: "Kick prediction request was blocked." });
  }

  let nestedPrediction: unknown;
  const data = record.data;
  if (data !== null && data !== undefined) {
    nestedPrediction = requireRecord(data).prediction;
  }

  const prediction = nestedPrediction ?? record.prediction;
  if (prediction === null || prediction === undefined) {
    return null;
  }
  return prediction;
}

function validatePrediction(value: unknown): ValidKickPrediction {
  const record = requireRecord(value);
  return {
    id: requireString(record, "id", { nonEmpty: true }),
    channel_id: requireNumber(record, "channel_id"),
    title: requireString(record, "title"),
    outcomes: requireArray(record, "outcomes").map(validateOutcome),
    duration: requireNumber(record, "duration"),
    created_at: readNullableString(record, "created_at"),
    updated_at: readNullableString(record, "updated_at"),
    locked_at: readNullableString(record, "locked_at"),
    state: requireString(record, "state", { nonEmpty: true }),
    winning_outcome_id: readNullableString(record, "winning_outcome_id")
  };
}

function validateOutcome(value: unknown): ValidKickOutcome {
  const record = requireRecord(value);
  return {
    id: requireString(record, "id", { nonEmpty: true }),
    title: requireString(record, "title"),
    total_vote_amount: requireNumber(record, "total_vote_amount"),
    vote_count: requireNumber(record, "vote_count"),
    return_rate: readOptionalNumber(record, "return_rate", 0),
    top_users: readOptionalArray(record, "top_users").map(validateTopUser)
  };
}

function validateTopUser(value: unknown): ValidKickTopUser {
  const record = requireRecord(value);
  return {
    id: requireNumber(record, "id"),
    username: requireString(record, "username", { nonEmpty: true }),
    amount: requireNumber(record, "amount")
  };
}

function requireRecord(value: unknown): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throwMalformedResponse();
  }
  return value as Record<string, unknown>;
}

function requireArray(record: Record<string, unknown>, key: string): unknown[] {
  const value = record[key];
  if (!Array.isArray(value)) {
    throwMalformedResponse();
  }
  return value;
}

function readOptionalArray(record: Record<string, unknown>, key: string): unknown[] {
  const value = record[key];
  if (value === null || value === undefined) {
    return [];
  }
  if (!Array.isArray(value)) {
    throwMalformedResponse();
  }
  return value;
}

function requireString(
  record: Record<string, unknown>,
  key: string,
  options: { nonEmpty?: boolean } = {}
): string {
  const value = record[key];
  if (typeof value !== "string") {
    throwMalformedResponse();
  }
  if (options.nonEmpty && value.trim() === "") {
    throwMalformedResponse();
  }
  return value;
}

function readNullableString(record: Record<string, unknown>, key: string): string | null {
  const value = record[key];
  if (value === null || value === undefined) {
    return null;
  }
  if (typeof value !== "string") {
    throwMalformedResponse();
  }
  return nullableString(value);
}

function requireNumber(record: Record<string, unknown>, key: string): number {
  const value = record[key];
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throwMalformedResponse();
  }
  return value;
}

function readOptionalNumber(
  record: Record<string, unknown>,
  key: string,
  defaultValue: number
): number {
  const value = record[key];
  if (value === null || value === undefined) {
    return defaultValue;
  }
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throwMalformedResponse();
  }
  return value;
}

function throwMalformedResponse(): never {
  throw new ApiClientError(502, { detail: "Kick prediction response was malformed." });
}

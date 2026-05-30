import { ApiClientError } from "@/lib/api-client";
import type { Prediction, PredictionOutcome, PredictionTopUser } from "@/types/api";

const KICK_API_BASE = "https://kick.com/api/v2";
const MAX_SLUG_LENGTH = 160;

type Fetcher = typeof fetch;

type KickTopUser = {
  id?: number;
  username?: string;
  amount?: number;
};

type KickOutcome = {
  id?: string;
  title?: string;
  total_vote_amount?: number;
  vote_count?: number;
  return_rate?: number;
  top_users?: KickTopUser[] | null;
};

type KickPrediction = {
  id?: string;
  channel_id?: number;
  title?: string;
  outcomes?: KickOutcome[] | null;
  duration?: number;
  created_at?: string | null;
  updated_at?: string | null;
  locked_at?: string | null;
  state?: string;
  winning_outcome_id?: string | null;
};

type KickPredictionEnvelope = {
  prediction?: KickPrediction | null;
  data?: { prediction?: KickPrediction | null } | null;
  error?: string | null;
};

export async function fetchKickPrediction(
  slug: string,
  fetchImpl: Fetcher = fetch,
  signal?: AbortSignal
): Promise<Prediction> {
  const normalized = slug.trim().toLowerCase();
  if (!normalized || normalized.length > MAX_SLUG_LENGTH) {
    throw new ApiClientError(404, { detail: "Invalid channel slug." });
  }

  await ensureChannelExists(normalized, fetchImpl, signal);
  return fetchLatestPrediction(normalized, fetchImpl, signal);
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

  let envelope: KickPredictionEnvelope;
  try {
    envelope = (await response.json()) as KickPredictionEnvelope;
  } catch {
    throw new ApiClientError(502, { detail: "Kick prediction response was malformed." });
  }

  if (envelope.error && envelope.error.trim() !== "") {
    throw new ApiClientError(502, { detail: "Kick prediction request was blocked." });
  }

  const raw = envelope.data?.prediction ?? envelope.prediction;
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

function normalizePrediction(raw: KickPrediction): Prediction {
  const winningOutcomeId = nullableString(raw.winning_outcome_id);
  const outcomes = (raw.outcomes ?? []).map((outcome) =>
    normalizeOutcome(outcome, winningOutcomeId)
  );

  const totalPoints = outcomes.reduce((sum, outcome) => sum + outcome.totalVoteAmount, 0);
  const totalVotes = outcomes.reduce((sum, outcome) => sum + outcome.voteCount, 0);

  for (const outcome of outcomes) {
    outcome.pointShare = totalPoints > 0 ? outcome.totalVoteAmount / totalPoints : 0;
  }

  return {
    id: raw.id ?? "",
    channelId: raw.channel_id ?? 0,
    title: raw.title ?? "",
    durationSeconds: raw.duration ?? 0,
    state: raw.state ?? "",
    winningOutcomeId,
    createdAt: nullableString(raw.created_at),
    lockedAt: nullableString(raw.locked_at),
    updatedAt: nullableString(raw.updated_at),
    totalPoints,
    totalVotes,
    outcomes
  };
}

function normalizeOutcome(
  outcome: KickOutcome,
  winningOutcomeId: string | null
): PredictionOutcome {
  const id = outcome.id ?? "";
  return {
    id,
    title: outcome.title ?? "",
    totalVoteAmount: outcome.total_vote_amount ?? 0,
    voteCount: outcome.vote_count ?? 0,
    returnRate: outcome.return_rate ?? 0,
    pointShare: 0,
    isWinner: winningOutcomeId !== null && id === winningOutcomeId,
    topUsers: (outcome.top_users ?? []).map(normalizeTopUser)
  };
}

function normalizeTopUser(user: KickTopUser): PredictionTopUser {
  return {
    id: user.id ?? 0,
    username: user.username ?? "",
    amount: user.amount ?? 0
  };
}

function nullableString(value: string | null | undefined): string | null {
  if (value === null || value === undefined) {
    return null;
  }
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
}

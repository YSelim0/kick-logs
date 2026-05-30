import { describe, expect, it, vi } from "vitest";

import { fetchKickPrediction } from "@/features/prediction/kick-prediction-client";
import { ApiClientError } from "@/lib/api-client";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" }
  });
}

function channelOk(): Response {
  return jsonResponse({ id: 1, slug: "nuriben" });
}

function predictionEnvelope(overrides: Record<string, unknown> = {}) {
  return {
    data: {
      prediction: {
        id: "pred-1",
        channel_id: 12440103,
        title: "mac bitis suresi",
        duration: 60,
        state: "RESOLVED",
        winning_outcome_id: "out-1",
        created_at: "2026-05-17T05:09:11Z",
        updated_at: "2026-05-17T05:13:10Z",
        locked_at: "2026-05-17T05:10:12Z",
        outcomes: [
          {
            id: "out-1",
            title: "Evet",
            total_vote_amount: 75,
            vote_count: 3,
            return_rate: 1.33,
            top_users: [{ id: 1, username: "alice", amount: 50 }]
          },
          {
            id: "out-2",
            title: "Hayir",
            total_vote_amount: 25,
            vote_count: 1,
            return_rate: 4,
            top_users: [{ id: 2, username: "bob", amount: 25 }]
          }
        ],
        ...overrides
      }
    },
    message: "Success"
  };
}

function fetchSequence(...responses: Response[]) {
  const fetchImpl = vi.fn();
  for (const response of responses) {
    fetchImpl.mockResolvedValueOnce(response);
  }
  return fetchImpl as unknown as typeof fetch;
}

describe("fetchKickPrediction", () => {
  it("normalizes the Kick payload and derives totals, share, and winner", async () => {
    const fetchImpl = fetchSequence(channelOk(), jsonResponse(predictionEnvelope()));

    const prediction = await fetchKickPrediction("NuriBen", fetchImpl);

    expect(prediction.title).toBe("mac bitis suresi");
    expect(prediction.durationSeconds).toBe(60);
    expect(prediction.totalPoints).toBe(100);
    expect(prediction.totalVotes).toBe(4);
    expect(prediction.outcomes[0].pointShare).toBe(0.75);
    expect(prediction.outcomes[0].isWinner).toBe(true);
    expect(prediction.outcomes[1].isWinner).toBe(false);
    expect(prediction.outcomes[0].topUsers).toEqual([{ id: 1, username: "alice", amount: 50 }]);
  });

  it("lowercases and trims the slug before requesting Kick", async () => {
    const fetchImpl = fetchSequence(channelOk(), jsonResponse(predictionEnvelope()));

    await fetchKickPrediction("  NuriBen  ", fetchImpl);

    expect(fetchImpl).toHaveBeenNthCalledWith(
      1,
      "https://kick.com/api/v2/channels/nuriben",
      expect.anything()
    );
    expect(fetchImpl).toHaveBeenNthCalledWith(
      2,
      "https://kick.com/api/v2/channels/nuriben/predictions/latest",
      expect.anything()
    );
  });

  it("caches channel validation for repeated requests with the same fetch implementation", async () => {
    const fetchImpl = fetchSequence(
      channelOk(),
      jsonResponse(predictionEnvelope({ title: "ilk veri" })),
      jsonResponse(predictionEnvelope({ title: "guncel veri" }))
    );

    await fetchKickPrediction("nuriben", fetchImpl);
    const prediction = await fetchKickPrediction("nuriben", fetchImpl);

    expect(prediction.title).toBe("guncel veri");
    expect(fetchImpl).toHaveBeenCalledTimes(3);
    expect(fetchImpl).toHaveBeenNthCalledWith(
      1,
      "https://kick.com/api/v2/channels/nuriben",
      expect.anything()
    );
    expect(fetchImpl).toHaveBeenNthCalledWith(
      2,
      "https://kick.com/api/v2/channels/nuriben/predictions/latest",
      expect.anything()
    );
    expect(fetchImpl).toHaveBeenNthCalledWith(
      3,
      "https://kick.com/api/v2/channels/nuriben/predictions/latest",
      expect.anything()
    );
  });

  it("does not divide by zero when there are no points", async () => {
    const envelope = predictionEnvelope({
      winning_outcome_id: null,
      outcomes: [
        {
          id: "out-1",
          title: "Evet",
          total_vote_amount: 0,
          vote_count: 0,
          return_rate: 0,
          top_users: []
        }
      ]
    });
    const fetchImpl = fetchSequence(channelOk(), jsonResponse(envelope));

    const prediction = await fetchKickPrediction("nuriben", fetchImpl);

    expect(prediction.totalPoints).toBe(0);
    expect(prediction.outcomes[0].pointShare).toBe(0);
  });

  it("tolerates a top-level prediction container", async () => {
    const fetchImpl = fetchSequence(
      channelOk(),
      jsonResponse({ prediction: predictionEnvelope().data.prediction })
    );

    const prediction = await fetchKickPrediction("nuriben", fetchImpl);

    expect(prediction.id).toBe("pred-1");
  });

  it("throws a 404 when the prediction is null", async () => {
    const fetchImpl = fetchSequence(channelOk(), jsonResponse({ data: { prediction: null } }));

    await expect(fetchKickPrediction("nuriben", fetchImpl)).rejects.toMatchObject({
      status: 404
    });
  });

  it("throws a 404 when the channel is missing", async () => {
    const fetchImpl = fetchSequence(jsonResponse({ message: "not found" }, 404));

    await expect(fetchKickPrediction("nuriben", fetchImpl)).rejects.toMatchObject({
      status: 404
    });
  });

  it("throws a non-404 when Kick blocks the request", async () => {
    const fetchImpl = fetchSequence(
      channelOk(),
      jsonResponse({ error: "Request blocked by security policy." })
    );

    await expect(fetchKickPrediction("nuriben", fetchImpl)).rejects.toMatchObject({
      status: 502
    });
  });

  it("throws a non-404 when the prediction shape is malformed", async () => {
    const fetchImpl = fetchSequence(
      channelOk(),
      jsonResponse(predictionEnvelope({ id: undefined }))
    );

    await expect(fetchKickPrediction("nuriben", fetchImpl)).rejects.toMatchObject({
      status: 502
    });
  });

  it("throws a non-404 when an outcome shape is malformed", async () => {
    const fetchImpl = fetchSequence(
      channelOk(),
      jsonResponse(
        predictionEnvelope({
          outcomes: [
            {
              id: "out-1",
              title: "Evet",
              total_vote_amount: "75",
              vote_count: 3,
              return_rate: 1.33,
              top_users: []
            }
          ]
        })
      )
    );

    await expect(fetchKickPrediction("nuriben", fetchImpl)).rejects.toMatchObject({
      status: 502
    });
  });

  it("throws a non-404 on a network failure", async () => {
    const fetchImpl = vi.fn().mockRejectedValue(new Error("network")) as unknown as typeof fetch;

    const error = await fetchKickPrediction("nuriben", fetchImpl).catch((caught) => caught);

    expect(error).toBeInstanceOf(ApiClientError);
    expect((error as ApiClientError).status).toBe(502);
  });
});

import { describe, expect, it } from "vitest";

import { ApiClientError } from "@/lib/api-client";
import { getUiErrorKey } from "./errors";

describe("localized UI errors", () => {
  it("uses operation context rather than server prose for authentication", () => {
    const error = new ApiClientError(401, { detail: "private internal details" });
    expect(getUiErrorKey(error, "login")).toBe("invalidCredentials");
    expect(getUiErrorKey(error, "adminRead")).toBe("sessionExpired");
  });

  it.each([
    [403, "forbidden"],
    [404, "notFound"],
    [409, "conflict"],
    [422, "invalidInput"],
    [429, "rateLimited"],
    [500, "unavailable"]
  ])("maps status %s without exposing raw exception text", (status, key) => {
    expect(getUiErrorKey(new ApiClientError(status, "secret trace"), "search")).toBe(key);
  });

  it("has safe network and unknown fallbacks", () => {
    expect(getUiErrorKey(new TypeError("Failed to fetch"), "search")).toBe("network");
    expect(getUiErrorKey(new Error("database trace"), "search")).toBe("unavailable");
  });
});

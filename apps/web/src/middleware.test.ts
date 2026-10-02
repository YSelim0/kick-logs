// @vitest-environment node
import { NextRequest, NextResponse } from "next/server";
import { describe, expect, it } from "vitest";
import { middleware, protectLocalizedResponse } from "./middleware";

describe("locale middleware", () => {
  it("initializes a secure host-only preference without redirecting", () => {
    const response = middleware(
      new NextRequest("https://kicklogs.net/search", {
        headers: { "accept-language": "de-DE,tr;q=0.7" }
      })
    );
    expect(response.status).toBe(200);
    expect(response.headers.get("location")).toBeNull();
    expect(response.cookies.get("kick_logs_locale")?.value).toBe("de");
    expect(response.headers.get("set-cookie")).toContain("Secure");
    expect(response.headers.get("set-cookie")).not.toContain("Domain=");
    expect(response.headers.get("cache-control")).toContain("no-store");
  });
  it("keeps a saved preference and does not refresh its cookie on every request", () => {
    const response = middleware(
      new NextRequest("http://localhost:3000/users", {
        headers: { cookie: "kick_logs_locale=tr", "accept-language": "de" }
      })
    );
    expect(response.headers.get("set-cookie")).toBeNull();
  });
  it.each([
    "/api/messages",
    "/_next/static/chunk.js",
    "/favicon.svg",
    "/language-flags/turkish-flag.svg"
  ])("leaves %s untouched", (path) => {
    const response = middleware(new NextRequest(`https://kicklogs.net${path}`));
    expect(response.headers.get("set-cookie")).toBeNull();
    expect(response.headers.get("cache-control")).toBeNull();
  });
  it("preserves existing Vary values", () => {
    const response = new NextResponse(null, { headers: { Vary: "RSC, Next-Router-State-Tree" } });
    protectLocalizedResponse(response);
    expect(response.headers.get("vary")).toBe(
      "RSC, Next-Router-State-Tree, Accept-Language, Cookie"
    );
  });
});

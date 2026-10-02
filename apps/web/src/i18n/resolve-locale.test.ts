import { describe, expect, it } from "vitest";

import { resolveLocale } from "./resolve-locale";

describe("locale negotiation", () => {
  it.each([
    ["de", "tr-TR,en;q=0.8", "de"],
    ["en", "de-DE", "en"],
    ["tr", null, "tr"],
    [undefined, "pt-BR, de-AT;q=0.8, tr;q=0.6", "de"],
    [undefined, "tr;q=0, en-US;q=0.8", "en"],
    [undefined, "de;q=0.4,tr;q=0.9", "tr"],
    [undefined, "de;q=0.8,tr;q=0.8", "de"],
    [undefined, "EN-us", "en"],
    [undefined, "TR-tr", "tr"],
    [undefined, "de-AT", "de"],
    [undefined, "pt-BR", "en"],
    [undefined, "*", "en"],
    [undefined, null, "en"],
    [undefined, "", "en"],
    ["../../other", "pt-BR", "en"],
    ["DE", "tr", "tr"],
    [undefined, "de;q=broken,tr;q=0.8", "tr"],
    [undefined, "de;q=2,tr;q=0.8", "tr"],
    [undefined, "de;q=-1,tr;q=0.8", "tr"],
    [undefined, "de;q=0.5,de;q=0.9,tr;q=0.7", "de"],
    [undefined, "garbage;;;, tr-TR;q=0.7", "tr"],
    [undefined, "de;q=0,tr;q=0", "en"]
  ])("resolves cookie %s and header %s to %s", (cookie, header, expected) => {
    expect(resolveLocale(cookie, header)).toBe(expected);
  });
});

import { describe, expect, it } from "vitest";

import { createFormatters } from "./format";

describe("localized display formats", () => {
  it("formats numbers and percentages without changing their values", () => {
    expect(createFormatters("en", "UTC").number(1234.5)).toBe("1,234.5");
    expect(createFormatters("de", "UTC").number(1234.5)).toBe("1.234,5");
    expect(createFormatters("tr", "UTC").percent(0.25)).toBe("%25");
    expect(createFormatters("en", "UTC").percent(0.25)).toBe("25%");
  });

  it("keeps UTC chart days independent of the display timezone", () => {
    const value = "2026-10-01T00:00:00Z";
    expect(createFormatters("en", "America/Los_Angeles").dayLabel(value)).toBe("Oct 1");
    expect(createFormatters("en", "UTC").dateTime(value)).toContain("00:00");
    expect(createFormatters("en", "Europe/Istanbul").dateTime(value)).toContain("03:00");
  });

  it("handles missing or invalid dates and zero bytes", () => {
    const format = createFormatters("en", "UTC");
    expect(format.dateTime(null)).toBe("—");
    expect(format.dateTime("not-a-date")).toBe("—");
    expect(format.bytes(0)).toBe("0 B");
    expect(format.bytes(1024)).toBe("1 KB");
  });
});

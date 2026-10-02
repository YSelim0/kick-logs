import { describe, expect, it } from "vitest";
import { buildChannelSubscribersExportUrl } from "./api";

describe("subscriber export locale", () => {
  it.each(["en", "tr", "de"] as const)(
    "adds %s only to TXT without changing the base or gift filter",
    (locale) => {
      const txt = new URL(
        buildChannelSubscribersExportUrl("Heaven", true, "txt", "https://example.test/api", locale)
      );
      expect(txt.pathname).toBe("/api/channels/Heaven/subscribers/export");
      expect(txt.searchParams.get("locale")).toBe(locale);
      expect(txt.searchParams.get("gift_only")).toBe("true");
      for (const format of ["json", "csv"] as const) {
        expect(
          buildChannelSubscribersExportUrl(
            "Heaven",
            true,
            format,
            "https://example.test/api",
            locale
          )
        ).toBe(
          buildChannelSubscribersExportUrl("Heaven", true, format, "https://example.test/api")
        );
      }
      expect(
        new URL(
          buildChannelSubscribersExportUrl("Heaven", false, "txt", "https://example.test/api")
        ).searchParams.has("locale")
      ).toBe(false);
    }
  );
});

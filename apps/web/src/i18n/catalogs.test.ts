import { describe, expect, it } from "vitest";
import { loadMessages, mergeMessages, type CatalogMessages } from "./catalogs";
import { validateCatalogs } from "../test/validate-catalogs";

describe("catalog contracts", () => {
  it("has identical valid ICU keys and arguments in all locales", async () => {
    // Validate raw catalogs, not the English fallback merged into runtime messages.
    const catalogs: CatalogMessages[] = [];
    for (const locale of ["en", "tr", "de"]) {
      const common = (await import(`./messages/${locale}/common.json`)).default;
      const publicMessages = (await import(`./messages/${locale}/public.json`)).default;
      const admin = (await import(`./messages/${locale}/admin.json`)).default;
      catalogs.push({ common, ...publicMessages, ...admin });
    }
    expect(() => validateCatalogs(catalogs)).not.toThrow();
  });
  it.each([
    [{ title: "Hello" }, {}],
    [{ title: "{name}" }, { title: "{other}" }],
    [{ title: "{count, number}" }, { title: "{count}" }],
    [{ title: "{count, plural, one {One}}" }, { title: "Broken {" }]
  ])("rejects inconsistent or malformed catalogs", (first, second) => {
    expect(() => validateCatalogs([first, second])).toThrow();
  });
  it("allows language-specific plural categories", () => {
    expect(() =>
      validateCatalogs([
        { title: "{n, plural, one {One} other {Many}}" },
        { title: "{n, plural, other {Several}}" }
      ])
    ).not.toThrow();
  });
  it("uses English leaves only when the selected catalog lacks them", () => {
    expect(
      mergeMessages({ actions: { close: "Close", save: "Save" } }, { actions: { close: "Kapat" } })
    ).toEqual({ actions: { close: "Kapat", save: "Save" } });
  });
  it("does not include admin catalogs in the public scope", async () => {
    const messages = await loadMessages("en", "public");
    expect(messages).not.toHaveProperty("adminShell");
  });
});

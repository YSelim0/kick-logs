import { describe, expect, it } from "vitest";

import { buildUserProfileHref, toKickProfileSlug } from "@/lib/kick-profile-slugs";

describe("kick profile slugs", () => {
  it("converts chat usernames with underscores to Kick profile slugs", () => {
    expect(toKickProfileSlug(" Example_User ")).toBe("example-user");
  });

  it("builds canonical local profile hrefs", () => {
    expect(buildUserProfileHref("Example_User")).toBe("/users/example-user");
    expect(buildUserProfileHref("")).toBeNull();
    expect(buildUserProfileHref(null)).toBeNull();
  });
});

import { describe, expect, it } from "vitest";

import {
  buildKickProfileUrl,
  buildUserProfileHref,
  toKickProfileSlug
} from "@/lib/kick-profile-slugs";

describe("kick profile slugs", () => {
  it("converts chat usernames with underscores to Kick profile slugs", () => {
    expect(toKickProfileSlug(" Example_User ")).toBe("example-user");
  });

  it("builds canonical local profile hrefs", () => {
    expect(buildUserProfileHref("Example_User")).toBe("/users/example-user");
    expect(buildUserProfileHref("")).toBeNull();
    expect(buildUserProfileHref(null)).toBeNull();
  });

  it("builds canonical Kick profile urls", () => {
    expect(buildKickProfileUrl("Example_User")).toBe("https://kick.com/example-user");
    expect(buildKickProfileUrl("")).toBeNull();
    expect(buildKickProfileUrl(null)).toBeNull();
  });
});

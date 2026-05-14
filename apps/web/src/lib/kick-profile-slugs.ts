export function toKickProfileSlug(value: string): string {
  return value.trim().toLowerCase().replace(/_/g, "-");
}

export function buildUserProfileHref(slug: string | null | undefined): string | null {
  if (!slug) {
    return null;
  }

  const profileSlug = toKickProfileSlug(slug);
  return profileSlug ? `/users/${encodeURIComponent(profileSlug)}` : null;
}

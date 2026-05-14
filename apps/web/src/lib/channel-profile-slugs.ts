export function buildChannelProfileHref(slug: string | null | undefined) {
  const normalized = slug?.trim();

  if (!normalized) {
    return null;
  }

  return `/channels/${encodeURIComponent(normalized)}`;
}

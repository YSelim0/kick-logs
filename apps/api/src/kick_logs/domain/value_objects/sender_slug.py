def normalize_kick_profile_slug(value: str | None) -> str | None:
    cleaned = _clean(value)
    return cleaned.replace("_", "-") if cleaned else None


def build_sender_lookup_terms(value: str | None) -> tuple[str, ...]:
    cleaned = _clean(value)
    if cleaned is None:
        return ()

    terms: list[str] = []
    for term in (
        cleaned,
        cleaned.replace("_", "-"),
        cleaned.replace("-", "_"),
    ):
        if term not in terms:
            terms.append(term)

    return tuple(terms)


def _clean(value: str | None) -> str | None:
    if value is None:
        return None

    cleaned = value.strip().lower()
    return cleaned or None

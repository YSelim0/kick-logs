from datetime import UTC, datetime

import pytest

from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.pagination import CursorPagination, MessageCursor
from kick_logs.domain.value_objects.roles import UserRole
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters


def test_search_filters_strip_empty_text() -> None:
    filters = MessageSearchFilters(sender=" yavuz ", channel="", q="   ")

    assert filters.sender == "yavuz"
    assert filters.channel is None
    assert filters.q is None
    assert filters.has_any_filter is True


def test_search_filters_treat_boolean_flags_as_active_filters() -> None:
    assert MessageSearchFilters(reply_only=True).has_any_filter is True
    assert MessageSearchFilters(emote_only=True).has_any_filter is True


def test_search_filters_reject_invalid_date_range() -> None:
    with pytest.raises(DomainError):
        MessageSearchFilters(
            start=datetime(2026, 5, 11, tzinfo=UTC),
            end=datetime(2026, 5, 10, tzinfo=UTC),
        )


def test_cursor_pagination_limits_range() -> None:
    cursor = MessageCursor(message_created_at=datetime(2026, 5, 10, tzinfo=UTC), message_id=100)

    pagination = CursorPagination(limit=100, cursor=cursor)

    assert pagination.cursor == cursor


@pytest.mark.parametrize("limit", [0, 101])
def test_cursor_pagination_rejects_out_of_range_limit(limit: int) -> None:
    with pytest.raises(DomainError):
        CursorPagination(limit=limit)


def test_super_admin_can_manage_admins() -> None:
    assert UserRole.SUPER_ADMIN.can_manage_admins is True
    assert UserRole.ADMIN.can_manage_admins is False

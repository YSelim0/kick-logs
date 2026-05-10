import pytest

from kick_logs.infrastructure.kick import ReconnectPolicy


def test_reconnect_policy_uses_exponential_backoff_with_cap() -> None:
    policy = ReconnectPolicy(initial_delay_seconds=1, max_delay_seconds=10, multiplier=2)

    assert [policy.delay_for_attempt(attempt) for attempt in range(1, 6)] == [
        1,
        2,
        4,
        8,
        10,
    ]


def test_reconnect_policy_rejects_invalid_attempts() -> None:
    with pytest.raises(ValueError):
        ReconnectPolicy().delay_for_attempt(0)

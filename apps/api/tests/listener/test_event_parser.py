import json

from kick_logs.infrastructure.kick import KickEventParser


def build_chat_event(payload: dict) -> str:
    return json.dumps(
        {
            "event": r"App\Events\ChatMessageEvent",
            "channel": "chatrooms.123.v2",
            "data": json.dumps(payload),
        }
    )


def representative_payload() -> dict:
    return {
        "id": "message-1",
        "chatroom_id": 123,
        "content": "hello [emote:37226:KEKW]",
        "type": "message",
        "created_at": "2026-05-10T01:02:03Z",
        "thread_parent_id": "thread-1",
        "sender": {
            "id": 456,
            "username": "Yavuz",
            "slug": "yavuz",
            "identity": {
                "color": "#fff600",
                "badges": [{"type": "moderator"}],
            },
        },
        "metadata": {
            "message_ref": "ref-1",
            "original_sender": {"username": "Other"},
            "original_message": {"content": "previous"},
        },
    }


def test_event_parser_extracts_representative_chat_payload() -> None:
    event = KickEventParser().parse(build_chat_event(representative_payload()))

    assert event is not None
    assert event.event == r"App\Events\ChatMessageEvent"
    assert event.channel == "chatrooms.123.v2"
    assert event.payload["id"] == "message-1"
    assert event.payload["chatroom_id"] == 123
    assert event.payload["content"] == "hello [emote:37226:KEKW]"
    assert event.payload["sender"]["username"] == "Yavuz"
    assert event.payload["sender"]["identity"]["badges"] == [{"type": "moderator"}]
    assert event.payload["metadata"]["message_ref"] == "ref-1"
    assert event.payload["thread_parent_id"] == "thread-1"


def test_event_parser_accepts_dict_envelopes_and_dict_data() -> None:
    payload = representative_payload()
    event = KickEventParser().parse(
        {
            "event": r"App\Events\ChatMessageEvent",
            "channel": "chatrooms.123.v2",
            "data": payload,
        }
    )

    assert event is not None
    assert event.payload is payload


def test_event_parser_ignores_non_chat_events() -> None:
    event = KickEventParser().parse(
        json.dumps({"event": "pusher:connection_established", "data": "{}"})
    )

    assert event is None


def test_event_parser_rejects_malformed_json_without_raising() -> None:
    event = KickEventParser().parse("not-json")

    assert event is None


def test_event_parser_rejects_incomplete_chat_payload() -> None:
    event = KickEventParser().parse(build_chat_event({"id": "message-1"}))

    assert event is None

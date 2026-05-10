from kick_logs.application.services.emote_parser import EmoteParser


def test_emote_parser_returns_empty_list_without_tokens() -> None:
    assert EmoteParser().parse("plain message") == []


def test_emote_parser_extracts_token_metadata_and_image_url() -> None:
    emotes = EmoteParser().parse("hello [emote:37226:KEKW]")

    assert len(emotes) == 1
    assert emotes[0].kick_emote_id == "37226"
    assert emotes[0].name == "KEKW"
    assert emotes[0].token == "[emote:37226:KEKW]"
    assert emotes[0].image_url == "https://files.kick.com/emotes/37226/fullsize"


def test_emote_parser_keeps_duplicate_occurrences() -> None:
    emotes = EmoteParser().parse("[emote:1:A] text [emote:1:A]")

    assert [emote.token for emote in emotes] == ["[emote:1:A]", "[emote:1:A]"]


def test_emote_parser_ignores_malformed_tokens() -> None:
    emotes = EmoteParser().parse("[emote:abc:BAD] [emote:123:] [emote:123:OK]")

    assert [emote.name for emote in emotes] == ["OK"]

import re

from kick_logs.domain.entities import Emote


class EmoteParser:
    _emote_pattern = re.compile(r"\[emote:(?P<id>\d+):(?P<name>[^\]]+)\]")

    def parse(self, content: str) -> list[Emote]:
        return [
            Emote(
                kick_emote_id=match.group("id"),
                name=match.group("name"),
                token=match.group(0),
            )
            for match in self._emote_pattern.finditer(content)
        ]

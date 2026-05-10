from dataclasses import dataclass

from kick_logs.domain.exceptions import DomainError


@dataclass(frozen=True, slots=True)
class Emote:
    kick_emote_id: str
    name: str
    token: str

    def __post_init__(self) -> None:
        if not self.kick_emote_id.strip():
            raise DomainError("Emote id is required.")
        if not self.name.strip():
            raise DomainError("Emote name is required.")
        if not self.token.strip():
            raise DomainError("Emote token is required.")

    @property
    def image_url(self) -> str:
        return f"https://files.kick.com/emotes/{self.kick_emote_id}/fullsize"

    def to_dict(self) -> dict[str, str]:
        return {
            "id": self.kick_emote_id,
            "name": self.name,
            "token": self.token,
            "image_url": self.image_url,
        }

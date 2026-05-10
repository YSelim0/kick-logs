from typing import Protocol


class TokenService(Protocol):
    def create_access_token(self, user_id: int) -> str: ...

    def get_user_id(self, token: str) -> int | None: ...

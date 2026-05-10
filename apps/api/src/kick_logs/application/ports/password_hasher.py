from typing import Protocol


class PasswordHasher(Protocol):
    def hash(self, plain_password: str) -> str: ...

    def verify(self, plain_password: str, password_hash: str) -> bool: ...

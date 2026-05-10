class ApplicationError(Exception):
    """Base application-layer error."""


class AuthenticationFailedError(ApplicationError):
    """Raised when credentials or auth tokens are invalid."""


class PermissionDeniedError(ApplicationError):
    """Raised when a user lacks the required role."""


class DuplicateUserEmailError(ApplicationError):
    """Raised when creating a user with an existing email."""


class UserNotFoundError(ApplicationError):
    """Raised when a user cannot be found or is inactive."""

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


class ChannelResolutionError(ApplicationError):
    """Raised when Kick channel metadata cannot be resolved."""


class ChannelNotFoundError(ApplicationError):
    """Raised when a followed channel cannot be found."""


class MessageIngestionError(ApplicationError):
    """Raised when an incoming Kick chat message payload cannot be normalized."""

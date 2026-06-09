"""Exception hierarchy for the Melosviz SDK."""

from __future__ import annotations


class MelosvizSDKError(Exception):
    """Base exception for all Melosviz SDK errors."""

    def __init__(self, message: str) -> None:
        super().__init__(message)
        self.message = message

    def __str__(self) -> str:
        return self.message


class MelosvizAPIError(MelosvizSDKError):
    """Raised when the NanoVMS API returns a non-success status code."""

    def __init__(self, status_code: int, body: str, message: str | None = None) -> None:
        super().__init__(message or f"HTTP {status_code}: {body}")
        self.status_code = status_code
        self.body = body

    def __repr__(self) -> str:
        return (
            f"MelosvizAPIError(status_code={self.status_code}, "
            f"body={self.body!r})"
        )


class MelosvizValidationError(MelosvizSDKError):
    """Raised when a request payload fails client-side validation."""

    def __init__(self, message: str, field: str | None = None) -> None:
        super().__init__(message)
        self.field = field

    def __repr__(self) -> str:
        return f"MelosvizValidationError(message={self.message!r}, field={self.field!r})"

"""Factory helpers for creating pre-configured MelosvizClient instances."""

from __future__ import annotations

from melosviz_sdk.client import MelosvizClient


def create_client(base_url: str, *, timeout: float = 30.0) -> MelosvizClient:
    """Create a new :class:`MelosvizClient` pointing at the given NanoVMS API base URL.

    Args:
        base_url: The NanoVMS API base URL (e.g., ``http://localhost:8080``).
        timeout: Request timeout in seconds.

    Returns:
        A configured :class:`MelosvizClient` instance.

    Example:
        >>> client = create_client("http://localhost:8080")
        >>> client.health()
        {'status': 'ok'}
    """
    return MelosvizClient(base_url, timeout=timeout)

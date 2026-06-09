"""Melosviz Python SDK for NanoVMS.

Provides synchronous and asynchronous HTTP clients for interacting with
NanoVMS runtimes (WASM, gVisor, Firecracker) via the REST API.

Example:
    >>> from melosviz_sdk import create_client, MelosvizClient
    >>> client = create_client("http://localhost:8080")
    >>> vms = client.list_vms()
    >>> print(vms)
"""

from __future__ import annotations

__version__ = "0.1.0"

from melosviz_sdk.client import MelosvizClient
from melosviz_sdk.errors import MelosvizSDKError, MelosvizAPIError, MelosvizValidationError
from melosviz_sdk.factory import create_client

__all__ = [
    "MelosvizClient",
    "MelosvizSDKError",
    "MelosvizAPIError",
    "MelosvizValidationError",
    "create_client",
    "__version__",
]

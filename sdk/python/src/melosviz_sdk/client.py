"""Synchronous and asynchronous HTTP client for the NanoVMS REST API."""

from __future__ import annotations

import json
from typing import Any, TypeVar

import httpx

from melosviz_sdk.errors import MelosvizAPIError, MelosvizSDKError

T = TypeVar("T")

DEFAULT_TIMEOUT = 30.0


class MelosvizClient:
    """HTTP client for the NanoVMS REST API.

    Args:
        base_url: The NanoVMS API base URL (e.g., ``http://localhost:8080``).
        timeout: Request timeout in seconds.
    """

    def __init__(self, base_url: str, *, timeout: float = DEFAULT_TIMEOUT) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._client: httpx.Client | None = None

    def _get_client(self) -> httpx.Client:
        if self._client is None:
            self._client = httpx.Client(timeout=self._timeout)
        return self._client

    def _request(self, method: str, path: str, **kwargs: Any) -> Any:
        url = f"{self._base_url}/api/v1{path}"
        client = self._get_client()
        try:
            response = client.request(method, url, **kwargs)
        except httpx.RequestError as exc:
            raise MelosvizSDKError(f"Request failed: {exc}") from exc

        if not response.is_success:
            raise MelosvizAPIError(
                status_code=response.status_code,
                body=response.text,
                message=f"HTTP {response.status_code}: {response.text}",
            )

        try:
            return response.json()
        except json.JSONDecodeError as exc:
            raise MelosvizSDKError(f"Failed to decode JSON response: {exc}") from exc

    def list_vms(self) -> list[dict[str, Any]]:
        """List all virtual machines."""
        return self._request("GET", "/vms")

    def get_vm(self, vm_id: str) -> dict[str, Any]:
        """Get a single VM by ID."""
        return self._request("GET", f"/vms/{vm_id}")

    def list_sandboxes(self) -> list[dict[str, Any]]:
        """List all sandboxes."""
        return self._request("GET", "/sandboxes")

    def get_sandbox(self, sandbox_id: str) -> dict[str, Any]:
        """Get a single sandbox by ID."""
        return self._request("GET", f"/sandboxes/{sandbox_id}")

    def deploy(self, tier: int, config: dict[str, Any]) -> dict[str, Any]:
        """Deploy a workload to the specified tier.

        Args:
            tier: Isolation tier (1=WASM, 2=gVisor, 3=Firecracker).
            config: Deployment configuration payload.
        """
        return self._request("POST", "/deploy", json={"tier": tier, "config": config})

    def health(self) -> dict[str, Any]:
        """Check API health status."""
        return self._request("GET", "/health")

    def close(self) -> None:
        """Close the underlying HTTP client and release resources."""
        if self._client is not None:
            self._client.close()
            self._client = None

    def __enter__(self) -> MelosvizClient:
        return self

    def __exit__(self, *exc: Any) -> None:
        self.close()

    def __repr__(self) -> str:
        return f"MelosvizClient(base_url={self._base_url!r})"

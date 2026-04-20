#!/usr/bin/env python3
"""Minimal Chrome DevTools Protocol helpers for dedicated-browser automation."""

from __future__ import annotations

import asyncio
import json
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

import requests
import websockets


class BrowserCdpError(RuntimeError):
    """Raised when CDP operations fail."""


def fetch_browser_ws_url(version_url: str, timeout_seconds: int = 10) -> str:
    response = requests.get(version_url, timeout=timeout_seconds)
    response.raise_for_status()
    payload = response.json()
    websocket_url = payload.get("webSocketDebuggerUrl")
    if not isinstance(websocket_url, str) or not websocket_url:
        raise BrowserCdpError(f"Missing webSocketDebuggerUrl from {version_url}: {payload!r}")
    return websocket_url


def normalize_url(url: str) -> str:
    cleaned = (url or "").strip()
    if not cleaned:
        return ""
    while cleaned.endswith("/"):
        cleaned = cleaned[:-1]
    return cleaned


@dataclass
class PageTarget:
    target_id: str
    title: str
    url: str
    attached: bool = False


class BrowserCdpClient:
    """Sequential CDP client suitable for low-concurrency worker scripts."""

    def __init__(self, websocket_url: str) -> None:
        self.websocket_url = websocket_url
        self._ws: Optional[websockets.WebSocketClientProtocol] = None
        self._next_id = 1

    async def connect(self) -> None:
        self._ws = await websockets.connect(self.websocket_url, max_size=None)

    async def close(self) -> None:
        if self._ws is not None:
            await self._ws.close()
            self._ws = None

    async def __aenter__(self) -> "BrowserCdpClient":
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        await self.close()

    async def browser_call(self, method: str, params: Optional[Dict[str, Any]] = None, *, timeout_seconds: float = 15) -> Dict[str, Any]:
        if self._ws is None:
            raise BrowserCdpError("Browser websocket is not connected.")
        call_id = self._next_id
        self._next_id += 1
        await self._ws.send(json.dumps({"id": call_id, "method": method, "params": params or {}}))
        return await self._wait_for_response(call_id, timeout_seconds=timeout_seconds)

    async def _wait_for_response(self, call_id: int, *, timeout_seconds: float) -> Dict[str, Any]:
        if self._ws is None:
            raise BrowserCdpError("Browser websocket is not connected.")
        while True:
            raw = await asyncio.wait_for(self._ws.recv(), timeout=timeout_seconds)
            payload = json.loads(raw)
            if payload.get("id") != call_id:
                continue
            if "error" in payload:
                raise BrowserCdpError(f"CDP error for call {call_id}: {payload['error']}")
            result = payload.get("result")
            if not isinstance(result, dict):
                return {}
            return result

    async def list_page_targets(self) -> List[PageTarget]:
        result = await self.browser_call("Target.getTargets")
        rows: List[PageTarget] = []
        for item in result.get("targetInfos", []):
            if not isinstance(item, dict) or item.get("type") != "page":
                continue
            rows.append(
                PageTarget(
                    target_id=item.get("targetId", ""),
                    title=item.get("title", ""),
                    url=item.get("url", ""),
                    attached=bool(item.get("attached")),
                )
            )
        return rows

    async def create_target(self, url: str = "about:blank", *, background: bool = True) -> str:
        result = await self.browser_call(
            "Target.createTarget",
            {
                "url": url,
                "background": background,
            },
        )
        target_id = result.get("targetId")
        if not isinstance(target_id, str) or not target_id:
            raise BrowserCdpError(f"Target.createTarget returned no targetId: {result!r}")
        return target_id

    async def close_target(self, target_id: str) -> None:
        await self.browser_call("Target.closeTarget", {"targetId": target_id})

    async def attach(self, target_id: str) -> "TargetSession":
        result = await self.browser_call("Target.attachToTarget", {"targetId": target_id, "flatten": True})
        session_id = result.get("sessionId")
        if not isinstance(session_id, str) or not session_id:
            raise BrowserCdpError(f"Target.attachToTarget returned no sessionId: {result!r}")
        return TargetSession(self, session_id=session_id, target_id=target_id)


class TargetSession:
    """Attached CDP session for one page target."""

    def __init__(self, client: BrowserCdpClient, *, session_id: str, target_id: str) -> None:
        self.client = client
        self.session_id = session_id
        self.target_id = target_id

    async def call(self, method: str, params: Optional[Dict[str, Any]] = None, *, timeout_seconds: float = 15) -> Dict[str, Any]:
        if self.client._ws is None:
            raise BrowserCdpError("Browser websocket is not connected.")
        call_id = self.client._next_id
        self.client._next_id += 1
        await self.client._ws.send(
            json.dumps(
                {
                    "id": call_id,
                    "method": method,
                    "params": params or {},
                    "sessionId": self.session_id,
                }
            )
        )
        return await self.client._wait_for_response(call_id, timeout_seconds=timeout_seconds)

    async def detach(self) -> None:
        await self.client.browser_call("Target.detachFromTarget", {"sessionId": self.session_id})

    async def enable_basics(self) -> None:
        await self.call("Page.enable")
        await self.call("Runtime.enable")

    async def navigate(self, url: str, *, settle_seconds: float = 2.0, ready_timeout_seconds: float = 15) -> None:
        await self.call("Page.navigate", {"url": url}, timeout_seconds=max(ready_timeout_seconds, 15))
        await asyncio.sleep(max(0.1, settle_seconds))
        await self.wait_for_ready_state(timeout_seconds=ready_timeout_seconds)

    async def wait_for_ready_state(self, *, timeout_seconds: float = 15, poll_seconds: float = 0.5) -> str:
        deadline = asyncio.get_event_loop().time() + timeout_seconds
        last_state = ""
        while asyncio.get_event_loop().time() < deadline:
            last_state = await self.evaluate_string("document.readyState", timeout_seconds=max(3, poll_seconds + 2))
            if last_state == "complete":
                return last_state
            await asyncio.sleep(poll_seconds)
        return last_state

    async def evaluate(self, expression: str, *, timeout_seconds: float = 15, await_promise: bool = False) -> Any:
        result = await self.call(
            "Runtime.evaluate",
            {
                "expression": expression,
                "returnByValue": True,
                "awaitPromise": await_promise,
            },
            timeout_seconds=timeout_seconds,
        )
        runtime_result = result.get("result", {})
        if not isinstance(runtime_result, dict):
            return None
        return runtime_result.get("value")

    async def evaluate_string(self, expression: str, *, timeout_seconds: float = 15) -> str:
        value = await self.evaluate(expression, timeout_seconds=timeout_seconds)
        if isinstance(value, str):
            return value
        return ""

    async def capture_screenshot(self, *, timeout_seconds: float = 20) -> bytes:
        result = await self.call(
            "Page.captureScreenshot",
            {"format": "png", "fromSurface": True},
            timeout_seconds=timeout_seconds,
        )
        data = result.get("data")
        if not isinstance(data, str) or not data:
            raise BrowserCdpError(f"Page.captureScreenshot returned no data: {result!r}")
        import base64

        return base64.b64decode(data)

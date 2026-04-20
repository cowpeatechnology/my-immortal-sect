#!/usr/bin/env python3
"""Shared URL filtering rules for browser-driven knowledge harvest."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Optional, Sequence
from urllib.parse import urlparse

import requests


BLOCKED_VIDEO_HOST_SUFFIXES = {
    "youtube.com",
    "youtu.be",
    "tiktok.com",
    "douyin.com",
    "bilibili.com",
    "ixigua.com",
    "kuaishou.com",
    "weishi.qq.com",
    "v.qq.com",
    "youku.com",
    "iqiyi.com",
    "mgtv.com",
    "tv.sohu.com",
    "pptv.com",
    "1905.com",
    "vimeo.com",
    "dailymotion.com",
    "twitch.tv",
    "huya.com",
    "douyu.com",
    "acfun.cn",
    "nicovideo.jp",
}

BLOCKED_DOCUMENT_EXTENSIONS = (
    ".pdf",
    ".doc",
    ".docx",
    ".ppt",
    ".pptx",
    ".xls",
    ".xlsx",
    ".csv",
    ".tsv",
    ".odt",
    ".ods",
    ".odp",
    ".rtf",
    ".epub",
    ".mobi",
    ".azw",
)

BLOCKED_DOCUMENT_CONTENT_TYPE_PREFIXES = (
    "application/pdf",
    "application/msword",
    "application/vnd.openxmlformats-officedocument.",
    "application/vnd.ms-",
    "application/vnd.oasis.opendocument.",
    "text/csv",
)

REQUEST_HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/147.0.0.0 Safari/537.36"
    )
}


@dataclass
class UrlProbeResult:
    original_url: str
    final_url: str
    content_type: str
    skip_reason: Optional[str]


def normalize_hostname(hostname: str) -> str:
    return (hostname or "").strip().lower().strip(".")


def hostname_matches_suffix(hostname: str, suffixes: Sequence[str]) -> bool:
    normalized = normalize_hostname(hostname)
    if not normalized:
        return False
    for suffix in suffixes:
        normalized_suffix = normalize_hostname(suffix)
        if normalized == normalized_suffix or normalized.endswith(f".{normalized_suffix}"):
            return True
    return False


def normalize_content_type(content_type: str) -> str:
    return (content_type or "").split(";", 1)[0].strip().lower()


def blocked_video_host_reason(hostname: str) -> Optional[str]:
    if hostname_matches_suffix(hostname, tuple(BLOCKED_VIDEO_HOST_SUFFIXES)):
        return "blocked-video-host"
    return None


def blocked_document_extension_reason(url: str) -> Optional[str]:
    path = urlparse(url).path.lower()
    for extension in BLOCKED_DOCUMENT_EXTENSIONS:
        if path.endswith(extension):
            return "blocked-document-extension"
    return None


def blocked_document_content_type_reason(content_type: str) -> Optional[str]:
    normalized = normalize_content_type(content_type)
    if not normalized:
        return None
    for prefix in BLOCKED_DOCUMENT_CONTENT_TYPE_PREFIXES:
        if normalized.startswith(prefix):
            return "blocked-document-content-type"
    return None


def skip_reason_for_url(url: str, hostname: str = "") -> Optional[str]:
    parsed = urlparse(url)
    effective_hostname = hostname or parsed.hostname or ""
    reason = blocked_video_host_reason(effective_hostname)
    if reason:
        return reason
    return blocked_document_extension_reason(url)


def probe_url(url: str, *, timeout_seconds: int) -> UrlProbeResult:
    initial_reason = skip_reason_for_url(url)
    if initial_reason:
        return UrlProbeResult(
            original_url=url,
            final_url=url,
            content_type="",
            skip_reason=initial_reason,
        )

    try:
        with requests.get(
            url,
            headers=REQUEST_HEADERS,
            stream=True,
            allow_redirects=True,
            timeout=timeout_seconds,
        ) as response:
            final_url = str(response.url or url)
            content_type = str(response.headers.get("content-type") or "")
    except requests.RequestException:
        return UrlProbeResult(
            original_url=url,
            final_url=url,
            content_type="",
            skip_reason=None,
        )

    redirected_reason = skip_reason_for_url(final_url)
    if redirected_reason:
        return UrlProbeResult(
            original_url=url,
            final_url=final_url,
            content_type=content_type,
            skip_reason=redirected_reason,
        )

    content_type_reason = blocked_document_content_type_reason(content_type)
    return UrlProbeResult(
        original_url=url,
        final_url=final_url,
        content_type=content_type,
        skip_reason=content_type_reason,
    )

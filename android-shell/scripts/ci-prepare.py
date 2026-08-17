#!/usr/bin/env python3
"""Prepare android-shell for CI: applicationId, cleartext flag, optional icon."""
from __future__ import annotations

import os
import re
import shutil
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlparse
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[1]
APP_RES = ROOT / "app" / "src" / "main" / "res"


def valid_app_id(value: str) -> bool:
    return bool(re.fullmatch(r"[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+", value or ""))


def segment(part: str) -> str:
    cleaned = re.sub(r"[^a-z0-9_]", "", (part or "").lower().replace("-", "_"))
    if not cleaned:
        cleaned = "app"
    if cleaned[0].isdigit():
        cleaned = "x" + cleaned
    return cleaned


def app_id_from_url(url: str) -> str:
    host = urlparse(url).hostname or "app.local"
    parts = [segment(p) for p in host.split(".") if p]
    if len(parts) < 2:
        parts = ["com", "pake"] + parts
    else:
        parts = list(reversed(parts))
    aid = ".".join(parts)
    return aid if valid_app_id(aid) else "com.pake.app"


def resolve_app_id(identifier: str, url: str) -> str:
    raw = (identifier or "").strip().lower().replace("-", "_")
    if valid_app_id(raw):
        return raw
    return app_id_from_url(url)


def write_output(app_id: str, cleartext: str) -> None:
    dest = os.environ.get("GITHUB_OUTPUT")
    if not dest:
        print(f"app_id={app_id}")
        print(f"cleartext={cleartext}")
        return
    with open(dest, "a", encoding="utf-8") as fh:
        fh.write(f"app_id={app_id}\n")
        fh.write(f"cleartext={cleartext}\n")


def fetch_icon(src: str, dest: Path) -> bool:
    src = (src or "").strip()
    if not src:
        return False
    dest.parent.mkdir(parents=True, exist_ok=True)
    tmp = dest.parent / "_icon_src.bin"
    try:
        if src.startswith("http://") or src.startswith("https://"):
            req = Request(src, headers={"User-Agent": "pake-gui-android-ci"})
            with urlopen(req, timeout=30) as resp:
                tmp.write_bytes(resp.read())
        else:
            path = Path(src)
            if not path.is_file():
                print(f"icon not found: {src}", file=sys.stderr)
                return False
            shutil.copyfile(path, tmp)
    except Exception as exc:
        print(f"icon fetch failed: {exc}", file=sys.stderr)
        return False

    converted = False
    for cmd in (
        ["magick", str(tmp), "-resize", "192x192", str(dest)],
        ["convert", str(tmp), "-resize", "192x192", str(dest)],
    ):
        try:
            subprocess.run(cmd, check=True)
            converted = dest.is_file()
            if converted:
                break
        except (FileNotFoundError, subprocess.CalledProcessError):
            continue
    if not converted:
        shutil.copyfile(tmp, dest)
    tmp.unlink(missing_ok=True)
    if dest.is_file():
        round_dest = dest.with_name("ic_launcher_round.png")
        shutil.copyfile(dest, round_dest)
    return dest.is_file()


def apply_icon(src: str) -> None:
    mipmap = APP_RES / "mipmap-xxxhdpi"
    dest = mipmap / "ic_launcher.png"
    if not fetch_icon(src, dest):
        print("no custom icon; keeping default adaptive icon")
        return
    anydpi = APP_RES / "mipmap-anydpi-v26"
    if anydpi.exists():
        shutil.rmtree(anydpi, ignore_errors=True)
    for name in (
        "mipmap-mdpi",
        "mipmap-hdpi",
        "mipmap-xhdpi",
        "mipmap-xxhdpi",
        "mipmap-xxxhdpi",
    ):
        d = APP_RES / name
        d.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(dest, d / "ic_launcher.png")
        shutil.copyfile(dest, d / "ic_launcher_round.png")
    print(f"installed launcher icon: {dest}")


def main() -> int:
    url = os.environ.get("PAKE_START_URL", "").strip()
    identifier = os.environ.get("PAKE_IDENTIFIER", "").strip()
    icon = os.environ.get("PAKE_ICON", "").strip()
    if not url:
        print("PAKE_START_URL is required", file=sys.stderr)
        return 1
    app_id = resolve_app_id(identifier, url)
    scheme = (urlparse(url).scheme or "").lower()
    cleartext = "true" if scheme == "http" else "false"
    print(f"applicationId={app_id} cleartext={cleartext}")
    write_output(app_id, cleartext)
    apply_icon(icon)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

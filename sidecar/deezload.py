"""
DeezLoad Telegram Bot Automation Tool

Automates song downloading via the DeezLoad Telegram bot,
extracts metadata, fetches synced lyrics from LRCLIB,
and embeds them into FLAC files.

This module can run in two modes:

1. **CLI mode** (default) — interactive menu-driven flow, identical to
   the original `deezload.py` standalone script. Use for one-off tasks.

2. **Sidecar mode** (`--sidecar` flag) — runs as a long-lived JSON-RPC
   subprocess driven by the Lymuru Go backend. Communicates over
   stdin/stdout. Progress / events are emitted on stderr as JSON lines.

The Go backend starts this script as a sidecar; do not invoke
`--sidecar` manually.
"""

import asyncio
import inspect
import json
import os
import re
import sys
import time
import unicodedata
import uuid
from pathlib import Path
from typing import Any, Callable, Optional

import requests
from dotenv import load_dotenv
from mutagen.flac import FLAC
from telethon import TelegramClient
from telethon.tl.custom import InlineResults
from telethon.tl.types import DocumentAttributeAudio, DocumentAttributeFilename

load_dotenv()

API_ID: int = int(os.getenv("TELEGRAM_API_ID", "0"))
API_HASH: str = os.getenv("TELEGRAM_API_HASH", "")
PHONE: str = os.getenv("TELEGRAM_PHONE", "")
SESSION_NAME: str = "deezload_session"
BOT_USERNAME: str = "deezload2bot"
# DOWNLOAD_DIR can be overridden by the Go backend via the SETTINGS method.
DOWNLOAD_DIR: Path = Path(os.getenv("LYMURU_DOWNLOAD_DIR", "downloads"))
LRCLIB_BASE: str = "https://lrclib.net/api/get"
RESULT_TIMEOUT: int = 120  # seconds to wait for bot response

# Global handle to the long-lived Telegram client (sidecar mode only).
_sidecar_client: Optional[TelegramClient] = None
# Event used to wake the auth flow when the user submits a code.
_auth_event: Optional[asyncio.Event] = None
_auth_code_holder: dict[str, str] = {}


def emit_event(event: dict[str, Any]) -> None:
    """Write a JSON event line to stderr (sidecar mode)."""
    if not event.get("type"):
        event["type"] = "event"
    sys.stderr.write(json.dumps(event, ensure_ascii=False) + "\n")
    sys.stderr.flush()


def emit_progress(task_id: str, stage: str, phase: str, percent: float = 0.0) -> None:
    emit_event({
        "type": "event",
        "name": "progress",
        "task_id": task_id,
        "stage": stage,
        "phase": phase,
        "download_percent": percent,
    })


def emit_complete(task_id: str, files: list[str]) -> None:
    emit_event({
        "type": "event",
        "name": "complete",
        "task_id": task_id,
        "files": files,
    })


def emit_error(task_id: str, message: str) -> None:
    emit_event({
        "type": "event",
        "name": "error",
        "task_id": task_id,
        "error": message,
    })


def send_response(req_id: str, result: Any) -> None:
    sys.stdout.write(json.dumps({"id": req_id, "ok": True, "result": result}, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def send_error(req_id: str, message: str) -> None:
    sys.stdout.write(json.dumps({"id": req_id, "ok": False, "error": message}, ensure_ascii=False) + "\n")
    sys.stdout.flush()


# ---------------------------------------------------------------------------
# Telegram helpers
# ---------------------------------------------------------------------------


async def connect_telegram_interactive() -> TelegramClient:
    """Connect with an interactive code prompt (CLI / first-run mode)."""
    if not API_ID or not API_HASH:
        raise SystemExit("[ERROR] TELEGRAM_API_ID and TELEGRAM_API_HASH must be set in .env")

    client = TelegramClient(SESSION_NAME, API_ID, API_HASH)
    await client.start(phone=PHONE)
    print("[OK] Connected to Telegram")
    return client


async def connect_telegram_sidecar() -> TelegramClient:
    """Connect in sidecar mode.

    If a valid session already exists, the connection succeeds without
    prompting. Otherwise, this raises `SessionPasswordNeededError`-style
    behaviour: it emits an `auth_needed` event and waits for the user
    to submit a code via the SETTINGS stdin request.
    """
    if not API_ID or not API_HASH:
        raise RuntimeError("TELEGRAM_API_ID and TELEGRAM_API_HASH must be set in .env")

    client = TelegramClient(SESSION_NAME, API_ID, API_HASH)
    # Try to connect with the existing session first.
    await client.connect()
    if not await client.is_user_authorized():
        # Need to sign in. Emit auth_needed and wait.
        global _auth_event
        _auth_event = asyncio.Event()
        sent = await client.send_code_request(PHONE)
        emit_event({
            "type": "event",
            "name": "auth_needed",
            "phone": PHONE,
            "phone_code_hash": sent.phone_code_hash,
        })
        # Wait for the user to submit a code (the Go side will forward
        # it via the SETTINGS method).
        try:
            await asyncio.wait_for(_auth_event.wait(), timeout=300)
        except asyncio.TimeoutError:
            await client.disconnect()
            raise RuntimeError("Telegram authentication timed out after 5 minutes")
        code = _auth_code_holder.get("code", "").strip()
        if not code:
            await client.disconnect()
            raise RuntimeError("No auth code provided")
        try:
            await client.sign_in(PHONE, code, phone_code_hash=sent.phone_code_hash)
        except Exception as exc:
            await client.disconnect()
            raise RuntimeError(f"Telegram sign-in failed: {exc}")
        # Auth succeeded; let the Go side know the session is now valid.
        emit_event({"type": "event", "name": "auth_success"})
    return client


async def search_song(client: TelegramClient, artist: str, title: str) -> Optional[InlineResults]:
    query = f"{title} {artist}"
    if not _is_sidecar():
        print(f"[SEARCH] Sending inline query to @{BOT_USERNAME}: {query}")
    results = await client.inline_query(BOT_USERNAME, query)
    if not results:
        if not _is_sidecar():
            print("[WARN] No results returned by bot.")
        return None
    if not _is_sidecar():
        print(f"[OK] Got {len(results)} result(s)")
    return results


def display_results(results: InlineResults) -> None:
    print()
    print("─" * 50)
    for i, result in enumerate(results):
        title = getattr(result, "title", None) or "(no title)"
        description = getattr(result, "description", None) or ""
        print(f"  [{i + 1}] {title}")
        if description:
            print(f"      {description}")
    print("─" * 50)


async def select_and_send_result(
    client: TelegramClient,
    results: InlineResults,
    choice: int,
) -> Optional[object]:
    me = await client.get_me()
    recent = await client.get_messages("me", limit=1)
    last_id = recent[0].id if recent else 0
    if not _is_sidecar():
        print(f"[SELECT] Sending result #{choice + 1} to Saved Messages...")
    try:
        await results[choice].click(entity=me.id)
    except Exception as exc:
        print(f"[ERROR] Failed to send inline result: {exc}")
        return None

    if not _is_sidecar():
        print("[WAIT] Waiting for audio in Saved Messages...")
    for _ in range(RESULT_TIMEOUT // 2):
        await asyncio.sleep(2)
        messages = await client.get_messages("me", limit=5, min_id=last_id)
        for msg in messages:
            if msg.document:
                for attr in msg.document.attributes:
                    if isinstance(attr, (DocumentAttributeAudio, DocumentAttributeFilename)):
                        if not _is_sidecar():
                            print("[OK] Audio message found in Saved Messages")
                        return msg
            if msg.audio:
                if not _is_sidecar():
                    print("[OK] Audio message found in Saved Messages")
                return msg

    if not _is_sidecar():
        print("[TIMEOUT] No audio appeared in Saved Messages.")
    return None


async def download_audio(
    client: TelegramClient,
    message: object,
    task_id: Optional[str] = None,
    web_progress_callback: Optional[Callable[[int, int], None]] = None,
) -> Optional[Path]:
    """Download the audio file from a Telegram message."""
    DOWNLOAD_DIR.mkdir(parents=True, exist_ok=True)

    filename = "unknown_track"
    for attr in message.document.attributes:
        if isinstance(attr, DocumentAttributeFilename):
            filename = attr.file_name
            break
        if isinstance(attr, DocumentAttributeAudio):
            performer = attr.performer or "Unknown"
            attr_title = attr.title or "Unknown"
            filename = f"{performer} - {attr_title}"

    filename = re.sub(r'[<>:"/\\|?*]', "_", filename)
    if not Path(filename).suffix:
        filename += ".flac"

    dest = DOWNLOAD_DIR / filename
    file_size = message.document.size or 0
    size_mb = file_size / (1024 * 1024)
    if not _is_sidecar():
        print(f"[DOWNLOAD] Downloading: {filename} ({size_mb:.1f} MB)")
    if task_id:
        emit_progress(task_id, f"Downloading {filename}", "preparing", 0)

    start_time = time.time()
    last_pct = -1

    def progress_callback(received: int, total: int) -> None:
        nonlocal last_pct
        t = total or file_size or 1
        pct = received / t * 100
        if not _is_sidecar():
            bar_len = 30
            filled = int(bar_len * received // t)
            bar = "█" * filled + "░" * (bar_len - filled)
            elapsed = time.time() - start_time
            speed = received / elapsed / 1024 if elapsed > 0 else 0
            print(
                f"\r  [{bar}] {pct:5.1f}%  {received / 1048576:.1f}/{t / 1048576:.1f} MB  {speed:.0f} KB/s",
                end="",
                flush=True,
            )
        if web_progress_callback:
            web_progress_callback(received, t)
        if task_id and pct - last_pct >= 1.0:
            last_pct = pct
            emit_progress(task_id, f"Downloading {filename}", "downloading", pct)

    with open(dest, "wb") as f:
        await client.download_file(
            message.document,
            f,
            part_size_kb=512,
            progress_callback=progress_callback,
        )
    if not _is_sidecar():
        print()

    elapsed = time.time() - start_time
    if dest.exists() and dest.stat().st_size > 0:
        if not _is_sidecar():
            print(f"[OK] Saved to {dest} ({elapsed:.1f}s)")
        if task_id:
            emit_progress(task_id, f"Saved to {dest.name}", "finalizing", 100)
        return dest
    print("[ERROR] Download failed")
    return None


# ---------------------------------------------------------------------------
# Lyrics helpers
# ---------------------------------------------------------------------------


def search_lrclib(artist: str, title: str) -> tuple[Optional[str], bool]:
    try:
        resp = requests.get(
            LRCLIB_BASE,
            params={"artist_name": artist, "track_name": title},
            timeout=15,
        )
        resp.raise_for_status()
        data = resp.json()
        synced = data.get("syncedLyrics")
        if synced:
            return synced, True
        plain = data.get("plainLyrics")
        if plain:
            return plain, False
        return None, False
    except requests.RequestException as exc:
        print(f"[ERROR] LRCLIB request failed: {exc}")
        return None, False


def detect_language(text: str) -> Optional[str]:
    clean = re.sub(r"\[\d{2}:\d{2}\.\d{2,3}\]", "", text)
    ja_count = 0
    zh_count = 0
    ko_count = 0
    for ch in clean:
        name = unicodedata.name(ch, "")
        if "HIRAGANA" in name or "KATAKANA" in name:
            ja_count += 1
        elif "CJK" in name:
            zh_count += 1
        elif "HANGUL" in name:
            ko_count += 1
    if ja_count > 0:
        return "ja"
    if ko_count > 0:
        return "ko"
    if zh_count > 5:
        return "zh"
    return None


def detect_line_languages(text: str) -> set[str]:
    langs = set()
    for ch in text:
        name = unicodedata.name(ch, "")
        if "HIRAGANA" in name or "KATAKANA" in name:
            langs.add("ja")
        elif "HANGUL" in name:
            langs.add("ko")
        elif "CJK" in name:
            langs.add("zh")
    if "ja" in langs and "zh" in langs:
        langs.discard("zh")
    return langs


def romanize_lyrics(lyrics: str) -> Optional[str]:
    all_langs = detect_line_languages(re.sub(r"\[\d{2}:\d{2}\.\d{2,3}\]", "", lyrics))
    if not all_langs:
        return None

    converters: dict[str, Any] = {}
    if "ja" in all_langs:
        try:
            import pykakasi

            kks = pykakasi.kakasi()
            converters["ja"] = lambda line: " ".join(
                item["hepburn"] for item in kks.convert(line) if item["hepburn"].strip()
            )
        except ImportError:
            print("[WARN] pykakasi not installed — run: pip install pykakasi")

    if "zh" in all_langs:
        try:
            from pypinyin import lazy_pinyin

            converters["zh"] = lambda line: " ".join(lazy_pinyin(line))
        except ImportError:
            print("[WARN] pypinyin not installed — run: pip install pypinyin")

    if "ko" in all_langs:
        try:
            from korean_romanizer.romanizer import Romanizer

            converters["ko"] = lambda line: Romanizer(line).romanize()
        except ImportError:
            print("[WARN] korean-romanizer not installed — run: pip install korean-romanizer")

    if not converters:
        return None

    def convert_line(text: str) -> str:
        line_langs = detect_line_languages(text)
        for lang in ("ja", "ko", "zh"):
            if lang in line_langs and lang in converters:
                return converters[lang](text)
        return text

    output_lines = []
    for line in lyrics.splitlines():
        match = re.match(r"((?:\[\d{2}:\d{2}\.\d{2,3}\]\s*)+)(.*)", line)
        if match:
            timestamp = match.group(1)
            text = match.group(2).strip()
            if text:
                output_lines.append(f"{timestamp}{convert_line(text)}")
            else:
                output_lines.append(line)
        elif line.strip():
            output_lines.append(convert_line(line))
        else:
            output_lines.append(line)
    return "\n".join(output_lines)


def embed_lyrics(file_path: Path, lyrics: str) -> None:
    if file_path.suffix.lower() != ".flac":
        print(f"[SKIP] Lyrics embedding only supported for FLAC — got {file_path.suffix}")
        return
    audio = FLAC(str(file_path))
    audio["LYRICS"] = lyrics
    audio.save()


def save_lrc_file(file_path: Path, lyrics: str, is_synced: bool = True) -> None:
    ext = ".lrc" if is_synced else ".txt"
    out_path = file_path.with_suffix(ext)
    out_path.write_text(lyrics, encoding="utf-8")


def extract_metadata(file_path: Path) -> tuple[str, str]:
    if file_path.suffix.lower() == ".flac":
        try:
            audio = FLAC(str(file_path))
            artist = audio.get("artist", [None])[0]
            title = audio.get("title", [None])[0]
            if artist and title:
                return artist, title
        except Exception as exc:
            print(f"[WARN] Could not read FLAC tags: {exc}")
    stem = file_path.stem
    if " - " in stem:
        parts = stem.split(" - ", 1)
        return parts[0].strip(), parts[1].strip()
    return "Unknown", stem


async def send_link_to_bot(
    client: TelegramClient,
    link: str,
    timeout_seconds: int = RESULT_TIMEOUT,
) -> Optional[object]:
    bot = await client.get_entity(BOT_USERNAME)
    sent = await client.send_message(bot, link)
    sent_id = sent.id
    if not _is_sidecar():
        print("[WAIT] Waiting for audio reply from bot...")
    for _ in range(max(1, timeout_seconds // 2)):
        await asyncio.sleep(2)
        messages = await client.get_messages(bot, limit=5, min_id=sent_id)
        for msg in messages:
            if not msg.out and msg.document:
                for attr in msg.document.attributes:
                    if isinstance(attr, (DocumentAttributeAudio, DocumentAttributeFilename)):
                        return msg
            if not msg.out and msg.audio:
                return msg
    return None


# ---------------------------------------------------------------------------
# Sidecar mode helpers
# ---------------------------------------------------------------------------


def _is_sidecar() -> bool:
    return "--sidecar" in sys.argv[1:]


def results_to_json(results: InlineResults) -> list[dict[str, Any]]:
    out = []
    for i, r in enumerate(results):
        out.append({
            "index": i,
            "title": getattr(r, "title", None) or "(no title)",
            "description": getattr(r, "description", None) or "",
        })
    return out


async def sidecar_method_search(params: dict[str, Any]) -> dict[str, Any]:
    global _sidecar_client
    if _sidecar_client is None:
        _sidecar_client = await connect_telegram_sidecar()
    artist = params.get("artist", "")
    title = params.get("title", "")
    results = await search_song(_sidecar_client, artist, title)
    if not results:
        return {"results": [], "search_key": ""}
    key = uuid.uuid4().hex
    # Stash the search key in a module-level dict so download can look it up.
    _sidecar_searches[key] = results
    return {"results": results_to_json(results), "search_key": key}


async def sidecar_method_connect(_params: dict[str, Any]) -> dict[str, Any]:
    """Connect to Telegram proactively (triggers auth flow if needed)."""
    global _sidecar_client
    if _sidecar_client is None:
        _sidecar_client = await connect_telegram_sidecar()
    return {"status": "connected"}


_sidecar_searches: dict[str, InlineResults] = {}

# Pending lyrics choices: task_id → {file_path, original, romanized, is_synced}.
_pending_lyrics: dict[str, dict[str, Any]] = {}

# Whether to save a separate .lrc/.txt file alongside the audio.
# Controlled by the export_lrc_file setting from the Go backend.
_lyrics_export_enabled: bool = True


async def sidecar_method_download(params: dict[str, Any]) -> dict[str, Any]:
    global _sidecar_client
    if _sidecar_client is None:
        _sidecar_client = await connect_telegram_sidecar()
    task_id = params.get("task_id") or uuid.uuid4().hex
    search_key = params.get("search_key", "")
    choice = int(params.get("choice", 0))
    results = _sidecar_searches.get(search_key)
    if results is None:
        raise RuntimeError("search_key expired or unknown; please re-search")
    emit_progress(task_id, "Submitting track to DeezLoad", "preparing", 0)
    message = await select_and_send_result(_sidecar_client, results, choice)
    if not message:
        emit_error(task_id, "No audio received from bot")
        raise RuntimeError("no audio message received")
    file_path = await download_audio(_sidecar_client, message, task_id=task_id)
    if not file_path:
        emit_error(task_id, "Audio download failed")
        raise RuntimeError("download failed")
    # Process lyrics asynchronously (may present choice to user).
    await _process_lyrics_for_download(task_id, file_path)
    return {"task_id": task_id}


async def sidecar_method_download_link(params: dict[str, Any]) -> dict[str, Any]:
    global _sidecar_client
    if _sidecar_client is None:
        _sidecar_client = await connect_telegram_sidecar()
    task_id = params.get("task_id") or uuid.uuid4().hex
    link = params.get("url") or params.get("link", "")
    if not link:
        raise RuntimeError("url is required")
    emit_progress(task_id, f"Sending link to DeezLoad", "preparing", 0)
    message = await send_link_to_bot(_sidecar_client, link)
    if not message:
        emit_error(task_id, "No audio reply from bot")
        raise RuntimeError("no audio message received")
    file_path = await download_audio(_sidecar_client, message, task_id=task_id)
    if not file_path:
        emit_error(task_id, "Audio download failed")
        raise RuntimeError("download failed")
    await _process_lyrics_for_download(task_id, file_path)
    return {"task_id": task_id}


async def sidecar_method_choose_lyrics(params: dict[str, Any]) -> dict[str, Any]:
    """Apply the user's lyrics choice and complete the task."""
    task_id = params.get("task_id", "")
    choice = params.get("choice", "original")
    data = _pending_lyrics.pop(task_id, None)
    if data is None:
        return {"status": "error", "error": f"no pending lyrics for task {task_id}"}
    _apply_lyrics_choice(task_id, data, choice)
    return {"status": "ok", "task_id": task_id}


# ---------------------------------------------------------------------------
# Post-download lyrics helpers
# ---------------------------------------------------------------------------

async def _process_lyrics_for_download(task_id: str, file_path: Path) -> None:
    """Fetch lyrics after audio download, present choice if romanized, then embed."""
    meta_artist, meta_title = extract_metadata(file_path)
    emit_progress(task_id, f"Looking up lyrics for {meta_artist} — {meta_title}", "preparing", 0)
    lyrics, is_synced = search_lrclib(meta_artist, meta_title)
    if not lyrics:
        # No lyrics — complete with audio only.
        emit_complete(task_id, [str(file_path)])
        return
    romanized = romanize_lyrics(lyrics)
    if romanized:
        # Store choice data and let the user pick.
        _pending_lyrics[task_id] = {
            "file_path": file_path,
            "original": lyrics,
            "romanized": romanized,
            "is_synced": is_synced,
        }
        emit_progress(task_id, "Lyrics found — choose format", "choosing", 100)
        # Auto-choose original if user doesn't respond within 60 s.
        asyncio.create_task(_lyrics_choice_timeout(task_id))
        return
    # No romanization needed — embed original directly.
    embed_lyrics(file_path, lyrics)
    files = _build_lyrics_files(file_path, lyrics, is_synced)
    emit_complete(task_id, files)


async def _lyrics_choice_timeout(task_id: str):
    """Auto-choose original lyrics after 60 seconds of inactivity."""
    await asyncio.sleep(60)
    data = _pending_lyrics.pop(task_id, None)
    if data is None:
        return  # already handled
    _apply_lyrics_choice(task_id, data, "original")


def _apply_lyrics_choice(task_id: str, data: dict[str, Any], choice: str):
    """Embed the chosen lyrics variant and complete the task."""
    file_path = data["file_path"]
    lyrics = data["original"] if choice == "original" else data["romanized"]
    is_synced = data["is_synced"]
    embed_lyrics(file_path, lyrics)
    files = _build_lyrics_files(file_path, lyrics, is_synced)
    emit_complete(task_id, files)


def _build_lyrics_files(file_path: Path, lyrics: str, is_synced: bool) -> list[str]:
    """Return the list of output files (audio + optional .lrc)."""
    files = [str(file_path)]
    if _lyrics_export_enabled:
        save_lrc_file(file_path, lyrics, is_synced)
        ext = ".lrc" if is_synced else ".txt"
        files.append(str(file_path.with_suffix(ext)))
    return files


async def sidecar_method_add_lyrics(params: dict[str, Any]) -> dict[str, Any]:
    task_id = params.get("task_id") or uuid.uuid4().hex
    file_path = Path(params.get("file_path", ""))
    if not file_path.exists():
        raise RuntimeError(f"file not found: {file_path}")
    artist = params.get("artist") or ""
    title = params.get("title") or ""
    if not artist or not title:
        meta_artist, meta_title = extract_metadata(file_path)
        artist = artist or meta_artist
        title = title or meta_title
    emit_progress(task_id, f"Fetching lyrics for {artist} — {title}", "preparing", 0)
    lyrics, is_synced = search_lrclib(artist, title)
    if not lyrics:
        emit_error(task_id, "No lyrics found on LRCLIB")
        raise RuntimeError("no lyrics found")
    embed_lyrics(file_path, lyrics)
    files = _build_lyrics_files(file_path, lyrics, is_synced)
    emit_complete(task_id, files)
    return {"task_id": task_id, "files": files}


async def sidecar_method_embed_lrc(params: dict[str, Any]) -> dict[str, Any]:
    task_id = params.get("task_id") or uuid.uuid4().hex
    flac_path = Path(params.get("flac_path", ""))
    lrc_path = Path(params.get("lrc_path", ""))
    if not flac_path.exists() or not lrc_path.exists():
        raise RuntimeError("flac or lrc file not found")
    lyrics = lrc_path.read_text(encoding="utf-8")
    emit_progress(task_id, f"Embedding LRC into {flac_path.name}", "preparing", 0)
    embed_lyrics(flac_path, lyrics)
    files = [str(flac_path)]
    emit_complete(task_id, files)
    return {"task_id": task_id, "files": files}


async def sidecar_method_romanize_lrc(params: dict[str, Any]) -> dict[str, Any]:
    lrc_path = Path(params.get("lrc_path", ""))
    if not lrc_path.exists():
        raise RuntimeError(f"file not found: {lrc_path}")
    lyrics = lrc_path.read_text(encoding="utf-8")
    romanized = romanize_lyrics(lyrics)
    if not romanized:
        return {"romanized": None, "download_url": None, "message": "No CJK characters detected"}
    out_path = lrc_path.with_name(lrc_path.stem + "_romanized.lrc")
    out_path.write_text(romanized, encoding="utf-8")
    return {
        "romanized": romanized,
        "download_url": str(out_path),
        "message": "Romanized LRC written to disk",
    }


async def sidecar_method_extract_lrc(params: dict[str, Any]) -> dict[str, Any]:
    flac_path = Path(params.get("flac_path", ""))
    if not flac_path.exists():
        raise RuntimeError(f"file not found: {flac_path}")
    audio = FLAC(str(flac_path))
    lyrics = audio.get("LYRICS", [None])[0]
    if not lyrics:
        raise RuntimeError("no embedded lyrics in this FLAC")
    is_synced = bool(re.search(r"\[\d{2}:\d{2}\.\d{2,3}\]", lyrics))
    ext = ".lrc" if is_synced else ".txt"
    out_path = flac_path.with_suffix(ext)
    out_path.write_text(lyrics, encoding="utf-8")
    return {
        "lyrics": lyrics,
        "is_synced": is_synced,
        "output_url": str(out_path),
    }


def sidecar_method_ping(_params: dict[str, Any]) -> dict[str, Any]:
    return {"pong": True}


def sidecar_method_submit_auth(params: dict[str, Any]) -> dict[str, Any]:
    """Forward an auth code from the Go side to the running connect flow."""
    code = (params.get("code") or "").strip()
    if not code:
        return {"status": "error", "error": "code is empty"}
    if _auth_event is None:
        return {"status": "error", "error": "no auth flow is in progress"}
    _auth_code_holder["code"] = code
    _auth_event.set()
    return {"status": "ok"}


def sidecar_method_set_settings(params: dict[str, Any]) -> dict[str, Any]:
    global DOWNLOAD_DIR, _lyrics_export_enabled
    if "downloads_folder" in params and params["downloads_folder"]:
        DOWNLOAD_DIR = Path(params["downloads_folder"])
    if "export_lrc_file" in params:
        v = params["export_lrc_file"]
        if isinstance(v, bool):
            _lyrics_export_enabled = v
        elif isinstance(v, str):
            _lyrics_export_enabled = v.lower() not in ("0", "false", "no", "")
        else:
            _lyrics_export_enabled = bool(v)
    return {"status": "ok", "downloads_folder": str(DOWNLOAD_DIR)}


_SIDECAR_METHODS = {
    "ping": sidecar_method_ping,
    "connect": sidecar_method_connect,
    "search": sidecar_method_search,
    "download": sidecar_method_download,
    "download_link": sidecar_method_download_link,
    "choose_lyrics": sidecar_method_choose_lyrics,
    "add_lyrics": sidecar_method_add_lyrics,
    "embed_lrc": sidecar_method_embed_lrc,
    "romanize_lrc": sidecar_method_romanize_lrc,
    "extract_lrc": sidecar_method_extract_lrc,
    "submit_auth": sidecar_method_submit_auth,
    "set_settings": sidecar_method_set_settings,
}


async def _dispatch(req_id: str, method: str, params: dict[str, Any]) -> None:
    handler = _SIDECAR_METHODS.get(method)
    if handler is None:
        send_error(req_id, f"unknown method: {method}")
        return
    try:
        if inspect.iscoroutinefunction(handler):
            result = await handler(params or {})
        else:
            result = handler(params or {})
        send_response(req_id, result)
    except Exception as exc:
        send_error(req_id, f"{type(exc).__name__}: {exc}")


async def _sidecar_main() -> None:
    """Run the JSON-RPC sidecar loop reading from stdin.

    Reads stdin synchronously in a background thread and feeds the lines
    into the asyncio event loop via a queue. This avoids the broken
    ``loop.connect_read_pipe(sys.stdin)`` Proactor path that crashes on
    Python 3.14 (and is fragile on other versions too).
    """
    import queue
    import threading

    line_queue: "queue.Queue[bytes | None]" = queue.Queue()

    def _stdin_reader() -> None:
        try:
            for raw in sys.stdin.buffer:
                line_queue.put(raw)
        except Exception:
            pass
        finally:
            line_queue.put(None)  # sentinel for EOF

    reader_thread = threading.Thread(target=_stdin_reader, name="sidecar-stdin", daemon=True)
    reader_thread.start()

    loop = asyncio.get_running_loop()
    while True:
        # Pull the next line off the queue without blocking the loop.
        raw = await loop.run_in_executor(None, line_queue.get)
        if raw is None:
            return
        try:
            req = json.loads(raw.decode("utf-8").strip())
        except Exception:
            continue
        if not isinstance(req, dict):
            continue
        req_id = req.get("id", "")
        method = req.get("method", "")
        params = req.get("params") or {}
        if not isinstance(params, dict):
            params = {}
        # Dispatch each request as a background task so long-running
        # handlers (e.g. search waiting for Telegram auth) don't block
        # other requests (e.g. submit_auth) from being processed.
        asyncio.create_task(_dispatch(req_id, method, params))


# ---------------------------------------------------------------------------
# Original CLI main (interactive mode)
# ---------------------------------------------------------------------------


async def main() -> None:
    """Main entry point: prompt user, download song, embed lyrics."""
    print("=" * 50)
    print("  DeezLoad — Telegram Music Downloader + Lyrics")
    print("=" * 50)
    print()

    print("  [1] Search & download from DeezLoad bot")
    print("  [2] Use existing local FLAC file")
    print("  [3] Embed existing LRC into FLAC file")
    print("  [4] Romanize existing LRC file")
    print("  [5] Extract LRC from FLAC file")
    print("  [6] Download from Spotify/Deezer link")
    print()

    while True:
        try:
            mode = int(input("Select mode [1-6]: ").strip())
            if mode in (1, 2, 3, 4, 5, 6):
                break
            print("  Enter 1, 2, 3, 4, 5, or 6")
        except ValueError:
            print("  Enter a valid number")

    if mode == 5:
        flac_input = input("FLAC file path: ").strip().strip('"')
        flac_path = Path(flac_input)
        if not flac_path.exists():
            print(f"[ERROR] File not found: {flac_path}")
            sys.exit(1)
        if flac_path.suffix.lower() != ".flac":
            print(f"[ERROR] Expected .flac file, got {flac_path.suffix}")
            sys.exit(1)
        audio = FLAC(str(flac_path))
        lyrics = audio.get("LYRICS", [None])[0]
        if not lyrics:
            print("[ERROR] No embedded lyrics found in this FLAC file.")
            sys.exit(1)
        is_synced = bool(re.search(r"\[\d{2}:\d{2}\.\d{2,3}\]", lyrics))
        ext = ".lrc" if is_synced else ".txt"
        out_path = flac_path.with_suffix(ext)
        out_path.write_text(lyrics, encoding="utf-8")
        fmt = "synced" if is_synced else "plain"
        print(f"[OK] Extracted {fmt} lyrics to: {out_path.name}")
        return

    if mode == 3:
        flac_input = input("FLAC file path: ").strip().strip('"')
        flac_path = Path(flac_input)
        if not flac_path.exists():
            print(f"[ERROR] File not found: {flac_path}")
            sys.exit(1)
        if flac_path.suffix.lower() != ".flac":
            print(f"[ERROR] Expected .flac file, got {flac_path.suffix}")
            sys.exit(1)
        lrc_input = input("LRC file path:  ").strip().strip('"')
        lrc_path = Path(lrc_input)
        if not lrc_path.exists():
            print(f"[ERROR] File not found: {lrc_path}")
            sys.exit(1)
        lyrics = lrc_path.read_text(encoding="utf-8")
        if not lyrics.strip():
            print("[ERROR] LRC file is empty.")
            sys.exit(1)
        embed_lyrics(flac_path, lyrics)
        print(f"[DONE] Lyrics embedded into: {flac_path}")
        return

    if mode == 4:
        lrc_input = input("LRC file path: ").strip().strip('"')
        lrc_path = Path(lrc_input)
        if not lrc_path.exists():
            print(f"[ERROR] File not found: {lrc_path}")
            sys.exit(1)
        lyrics = lrc_path.read_text(encoding="utf-8")
        if not lyrics.strip():
            print("[ERROR] LRC file is empty.")
            sys.exit(1)
        romaji = romanize_lyrics(lyrics)
        if not romaji:
            print("[DONE] No CJK characters detected — nothing to romanize.")
            return
        lrc_path.write_text(romaji, encoding="utf-8")
        print(f"[DONE] Romanized LRC saved: {lrc_path}")
        return

    if mode == 6:
        link = input("Spotify/Deezer link: ").strip()
        if not link:
            print("[ERROR] A link is required.")
            sys.exit(1)
        client = await connect_telegram_interactive()
        try:
            message = await send_link_to_bot(client, link)
            if not message:
                print("[ABORT] No audio file received. Exiting.")
                return
            file_path = await download_audio(client, message)
            if not file_path:
                print("[ABORT] Download failed. Exiting.")
                return
            meta_artist, meta_title = extract_metadata(file_path)
            lyrics, is_synced = search_lrclib(meta_artist, meta_title)
            romaji_lyrics = None
            if lyrics:
                romaji_lyrics = romanize_lyrics(lyrics)
            if lyrics:
                # In CLI mode, default to original unless user wants romanized.
                chosen = lyrics
                if romaji_lyrics:
                    print()
                    print("  [1] Original lyrics")
                    print("  [2] Romanized lyrics")
                    while True:
                        c = input("LRC format [1-2]: ").strip()
                        if c == "1":
                            chosen = lyrics
                            break
                        if c == "2":
                            chosen = romaji_lyrics
                            break
                        print("  Enter 1 or 2")
                embed_lyrics(file_path, chosen)
                save_lrc_file(file_path, chosen, is_synced)
            else:
                print("[DONE] No lyrics to embed.")
            print(f"[DONE] File ready: {file_path}")
        finally:
            await client.disconnect()
        return

    if mode == 2:
        file_input = input("FLAC file path: ").strip().strip('"')
        file_path = Path(file_input)
        if not file_path.exists():
            print(f"[ERROR] File not found: {file_path}")
            sys.exit(1)
        if file_path.suffix.lower() != ".flac":
            print(f"[ERROR] Expected .flac file, got {file_path.suffix}")
            sys.exit(1)
        meta_artist, meta_title = extract_metadata(file_path)
        artist_input = input(f"Artist [{meta_artist}]: ").strip() or meta_artist
        title_input = input(f"Title  [{meta_title}]: ").strip() or meta_title
        lyrics, is_synced = search_lrclib(artist_input, title_input)
        romaji_lyrics = None
        if lyrics:
            romaji_lyrics = romanize_lyrics(lyrics)
            chosen = lyrics
            if romaji_lyrics:
                print()
                print("  [1] Original lyrics")
                print("  [2] Romanized lyrics")
                while True:
                    c = input("LRC format [1-2]: ").strip()
                    if c == "1":
                        chosen = lyrics
                        break
                    if c == "2":
                        chosen = romaji_lyrics
                        break
                    print("  Enter 1 or 2")
            embed_lyrics(file_path, chosen)
            save_lrc_file(file_path, chosen, is_synced)
        else:
            print("[DONE] No lyrics found.")
        print(f"[DONE] File ready: {file_path}")
        return

    # --- DeezLoad bot mode ---
    artist_input = input("Artist: ").strip()
    title_input = input("Title:  ").strip()
    if not artist_input or not title_input:
        print("[ERROR] Artist and title are required.")
        sys.exit(1)
    client = await connect_telegram_interactive()
    try:
        results = await search_song(client, artist_input, title_input)
        if not results:
            print("[ABORT] No results found. Exiting.")
            return
        display_results(results)
        while True:
            try:
                pick = int(input(f"\nSelect track [1-{len(results)}]: ").strip())
                if 1 <= pick <= len(results):
                    break
                print(f"  Enter a number between 1 and {len(results)}")
            except ValueError:
                print("  Enter a valid number")
        message = await select_and_send_result(client, results, pick - 1)
        if not message:
            print("[ABORT] No audio file received. Exiting.")
            return
        file_path = await download_audio(client, message)
        if not file_path:
            print("[ABORT] Download failed. Exiting.")
            return
        meta_artist, meta_title = extract_metadata(file_path)
        lyrics, is_synced = search_lrclib(meta_artist, meta_title)
        if not lyrics:
            if (meta_artist.lower() != artist_input.lower()
                    or meta_title.lower() != title_input.lower()):
                print("[RETRY] Trying LRCLIB with original input...")
                lyrics, is_synced = search_lrclib(artist_input, title_input)
        romaji_lyrics = None
        if lyrics:
            romaji_lyrics = romanize_lyrics(lyrics)
        if lyrics:
            chosen = lyrics
            if romaji_lyrics:
                print()
                print("  [1] Original lyrics")
                print("  [2] Romanized lyrics")
                while True:
                    c = input("LRC format [1-2]: ").strip()
                    if c == "1":
                        chosen = lyrics
                        break
                    if c == "2":
                        chosen = romaji_lyrics
                        break
                    print("  Enter 1 or 2")
            embed_lyrics(file_path, chosen)
            save_lrc_file(file_path, chosen, is_synced)
        else:
            print("[DONE] No lyrics to embed.")
        print(f"[DONE] File ready: {file_path}")
    finally:
        await client.disconnect()


if __name__ == "__main__":
    if "--sidecar" in sys.argv[1:]:
        asyncio.run(_sidecar_main())
    else:
        asyncio.run(main())
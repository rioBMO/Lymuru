"""
Lymuru API — thin web layer on top of deezload.py.

Telegram credentials and session are managed by deezload.py (run it once
to authenticate).  This backend simply reuses the same session file so
the browser UI never needs to handle Telegram auth flows.
"""

import asyncio
import os
import re
import shutil
import tempfile
import time
import uuid
from pathlib import Path
from typing import Any, Optional

from fastapi import FastAPI, File, Form, Header, HTTPException, Request, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles

import sys
import io

# Fix Windows encoding: force UTF-8 for stdout/stderr so CJK characters
# from deezload.py (Japanese, Chinese, Korean) don't crash with charmap.
if sys.platform == "win32":
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

import deezload  # noqa: E402

app = FastAPI(title="Lymuru API", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# ── Directories ──────────────────────────────────────────────────────
UPLOAD_DIR = Path(os.getenv("UPLOAD_DIR", "uploads"))
DOWNLOAD_DIR = Path(os.getenv("DOWNLOAD_DIR", "downloads"))
SESSION_DIR = Path(os.getenv("SESSION_DIR", "sessions"))
API_TOKEN = os.getenv("API_TOKEN", "")
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
DOWNLOAD_DIR.mkdir(parents=True, exist_ok=True)

# ── Auth middleware ───────────────────────────────────────────────────
@app.middleware("http")
async def auth_middleware(request: Request, call_next):
    if API_TOKEN and request.url.path.startswith("/api/") and request.method != "OPTIONS" and request.url.path not in ("/api/health", "/api/auth/check"):
        token = (request.headers.get("Authorization", "") or "").removeprefix("Bearer ")
        if token != API_TOKEN:
            return JSONResponse(status_code=401, content={"detail": "Invalid or missing API token"})
    return await call_next(request)

# ── Task progress store (in-memory) ─────────────────────────────────
_tasks: dict[str, dict[str, Any]] = {}


def _new_task() -> str:
    tid = uuid.uuid4().hex[:12]
    _tasks[tid] = {
        "stage": "Starting…", "done": False, "error": None, "files": [],
        "phase": "preparing",        # "preparing" | "downloading" | "finalizing"
        "download_percent": 0,       # 0–100  real download %
        "download_received": 0,      # bytes received
        "download_total": 0,         # bytes total
        "has_romanized": False, "waiting_for_choice": False,
        # internal data (not sent to client)
        "_lyrics": None, "_romanized": None, "_is_synced": False, "_file_path": None,
    }
    return tid


def _update(tid: str, stage: str, phase: str = "") -> None:
    if tid in _tasks:
        _tasks[tid]["stage"] = stage
        if phase:
            _tasks[tid]["phase"] = phase


def _update_download(tid: str, received: int, total: int) -> None:
    """Called from the download progress callback with real byte counts."""
    if tid in _tasks:
        _tasks[tid]["download_received"] = received
        _tasks[tid]["download_total"] = total
        _tasks[tid]["download_percent"] = round(received / total * 100, 1) if total else 0


def _finish(tid: str, files: list[str]) -> None:
    if tid in _tasks:
        _tasks[tid]["stage"] = "Complete"
        _tasks[tid]["phase"] = "complete"
        _tasks[tid]["done"] = True
        _tasks[tid]["files"] = files
        _tasks[tid]["waiting_for_choice"] = False


def _wait_for_choice(tid: str, file_path: Path, lyrics: str, romanized: str, is_synced: bool) -> None:
    """Pause the task and wait for the user to choose lyrics format."""
    if tid in _tasks:
        _tasks[tid]["stage"] = "Choose lyrics format"
        _tasks[tid]["phase"] = "choosing"
        _tasks[tid]["has_romanized"] = True
        _tasks[tid]["waiting_for_choice"] = True
        _tasks[tid]["_lyrics"] = lyrics
        _tasks[tid]["_romanized"] = romanized
        _tasks[tid]["_is_synced"] = is_synced
        _tasks[tid]["_file_path"] = str(file_path)


def _fail(tid: str, error: str) -> None:
    if tid in _tasks:
        _tasks[tid]["stage"] = f"Error: {error}"
        _tasks[tid]["done"] = True
        _tasks[tid]["error"] = error


# ── Telegram client singleton ────────────────────────────────────────
_client = None
_client_lock = asyncio.Lock()


async def get_client():
    global _client
    async with _client_lock:
        if _client is None or not _client.is_connected():
            session_path = SESSION_DIR / "deezload_session"
            if not Path(f"{session_path}.session").exists():
                # fallback to root session
                if Path("deezload_session.session").exists():
                    SESSION_DIR.mkdir(parents=True, exist_ok=True)
                    shutil.copy2("deezload_session.session", f"{session_path}.session")
                else:
                    raise HTTPException(
                        status_code=503,
                        detail="No Telegram session found. Run deezload.py first to authenticate.",
                    )
            from telethon import TelegramClient
            _client = TelegramClient(
                str(session_path),
                deezload.API_ID,
                deezload.API_HASH,
            )
            await _client.connect()
            if not await _client.is_user_authorized():
                raise HTTPException(
                    status_code=503,
                    detail="Telegram session expired. Run deezload.py to re-authenticate.",
                )
        return _client


# ── Health & Status ──────────────────────────────────────────────────

@app.get("/api/health")
async def health():
    return {"status": "ok"}


@app.get("/api/auth/check")
async def auth_check(authorization: Optional[str] = Header(None)):
    if API_TOKEN:
        token = (authorization or "").removeprefix("Bearer ")
        if token != API_TOKEN:
            raise HTTPException(status_code=401, detail="Invalid or missing API token")
    return {"valid": True}


@app.get("/api/telegram/status")
async def telegram_status():
    configured = bool(deezload.API_ID and deezload.API_HASH)
    if not configured:
        return {"configured": False, "authorized": False, "message": "Not configured"}
    try:
        client = await get_client()
        authorized = await client.is_user_authorized()
        return {
            "configured": True,
            "authorized": authorized,
            "message": "Connected" if authorized else "Session expired",
        }
    except HTTPException as exc:
        return {"configured": True, "authorized": False, "message": exc.detail}
    except Exception as exc:
        return {"configured": True, "authorized": False, "message": str(exc)}


# ── Progress polling ─────────────────────────────────────────────────

@app.get("/api/downloads/progress/{task_id}")
async def get_progress(task_id: str):
    task = _tasks.get(task_id)
    if not task:
        raise HTTPException(status_code=404, detail="Task not found")
    # Return only public fields (strip internal _-prefixed keys)
    return {k: v for k, v in task.items() if not k.startswith("_")}


# ── Tab 1: Search & Download ─────────────────────────────────────────

@app.post("/api/search")
async def search_song(artist: str = Form(...), title: str = Form(...)):
    """Search DeezLoad bot and return results."""
    try:
        client = await get_client()
        results = await deezload.search_song(client, artist, title)
        if not results:
            return {"results": []}
        items = []
        for i, r in enumerate(results):
            items.append({
                "index": i,
                "title": getattr(r, "title", "(no title)") or "(no title)",
                "description": getattr(r, "description", "") or "",
            })
        # Store results in memory for selection
        _search_cache[f"{artist}|{title}"] = results
        return {"results": items, "search_key": f"{artist}|{title}"}
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))


_search_cache: dict[str, Any] = {}


@app.post("/api/downloads/choose")
async def download_chosen(
    search_key: str = Form(...),
    choice: int = Form(...),
    artist: str = Form(""),
    title: str = Form(""),
):
    """Download a chosen search result (Tab 1)."""
    results = _search_cache.get(search_key)
    if not results:
        raise HTTPException(status_code=400, detail="Search results expired. Search again.")

    tid = _new_task()

    async def _run():
        try:
            client = await get_client()

            _update(tid, "Sending selection to bot…", "preparing")
            message = await deezload.select_and_send_result(client, results, choice)
            if not message:
                _fail(tid, "No audio file received from bot")
                return

            _update(tid, "Downloading audio…", "downloading")
            file_path = await deezload.download_audio(
                client, message,
                web_progress_callback=lambda r, t: _update_download(tid, r, t),
            )
            if not file_path:
                _fail(tid, "Download failed")
                return

            _update(tid, "Extracting metadata…", "finalizing")
            meta_artist, meta_title = deezload.extract_metadata(file_path)

            _update(tid, "Searching lyrics…")
            lyrics, is_synced = deezload.search_lrclib(meta_artist, meta_title)
            if not lyrics and (meta_artist.lower() != artist.lower() or meta_title.lower() != title.lower()):
                lyrics, is_synced = deezload.search_lrclib(artist, title)

            romaji_lyrics = None
            if lyrics:
                _update(tid, "Romanizing lyrics…")
                romaji_lyrics = deezload.romanize_lyrics(lyrics)

            if lyrics and romaji_lyrics:
                _wait_for_choice(tid, file_path, lyrics, romaji_lyrics, is_synced)
            elif lyrics:
                _update(tid, "Embedding lyrics…")
                deezload.embed_lyrics(file_path, lyrics)
                _update(tid, "Saving LRC file…")
                deezload.save_lrc_file(file_path, lyrics, is_synced)
                files = [str(file_path)]
                lrc_path = file_path.with_suffix(".lrc" if is_synced else ".txt")
                if lrc_path.exists():
                    files.append(str(lrc_path))
                _finish(tid, files)
            else:
                _finish(tid, [str(file_path)])
        except Exception as exc:
            _fail(tid, str(exc))

    asyncio.create_task(_run())
    return {"task_id": tid}


@app.post("/api/downloads/choose-lyrics")
async def choose_lyrics(
    task_id: str = Form(...),
    lyrics_choice: str = Form(...),  # "original" or "romanized"
):
    """Finalize a download by embedding the user's chosen lyrics format."""
    task = _tasks.get(task_id)
    if not task:
        raise HTTPException(status_code=404, detail="Task not found")
    if not task.get("waiting_for_choice"):
        raise HTTPException(status_code=400, detail="Task is not waiting for lyrics choice")

    file_path = Path(task["_file_path"])
    lyrics = task["_lyrics"]
    romanized = task["_romanized"]
    is_synced = task["_is_synced"]

    chosen = romanized if lyrics_choice == "romanized" else lyrics

    _update(task_id, "Embedding lyrics…", "finalizing")
    deezload.embed_lyrics(file_path, chosen)
    _update(task_id, "Saving LRC file…")
    deezload.save_lrc_file(file_path, chosen, is_synced)

    files = [str(file_path)]
    lrc_path = file_path.with_suffix(".lrc" if is_synced else ".txt")
    if lrc_path.exists():
        files.append(str(lrc_path))

    _finish(task_id, files)
    return {"status": "ok", "task_id": task_id}


# ── Tab 2: Add Lyrics to FLAC ────────────────────────────────────────

@app.post("/api/lyrics/add")
async def add_lyrics(
    file: UploadFile = File(...),
    artist: str = Form(""),
    title: str = Form(""),
):
    """Upload a FLAC, search lyrics, embed them (Tab 2)."""
    tid = _new_task()
    UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
    dest = UPLOAD_DIR / file.filename
    with open(dest, "wb") as f:
        content = await file.read()
        f.write(content)

    async def _run():
        try:
            _update(tid, "Extracting metadata…", "preparing")
            meta_artist, meta_title = deezload.extract_metadata(dest)
            use_artist = artist or meta_artist
            use_title = title or meta_title

            _update(tid, "Searching lyrics…")
            lyrics, is_synced = deezload.search_lrclib(use_artist, use_title)

            romaji_lyrics = None
            if lyrics:
                _update(tid, "Romanizing lyrics…")
                romaji_lyrics = deezload.romanize_lyrics(lyrics)

            if lyrics and romaji_lyrics:
                _wait_for_choice(tid, dest, lyrics, romaji_lyrics, is_synced)
            elif lyrics:
                _update(tid, "Embedding lyrics…", "finalizing")
                deezload.embed_lyrics(dest, lyrics)
                _update(tid, "Saving LRC file…")
                deezload.save_lrc_file(dest, lyrics, is_synced)
                files = [str(dest)]
                lrc_path = dest.with_suffix(".lrc" if is_synced else ".txt")
                if lrc_path.exists():
                    files.append(str(lrc_path))
                _finish(tid, files)
            else:
                _finish(tid, [str(dest)])
        except Exception as exc:
            _fail(tid, str(exc))

    asyncio.create_task(_run())
    return {"task_id": tid, "meta_artist": artist, "meta_title": title}


# ── Tab 3: Embed LRC into FLAC ───────────────────────────────────────

@app.post("/api/lyrics/embed")
async def embed_lrc(
    flac_file: UploadFile = File(...),
    lrc_file: UploadFile = File(...),
):
    """Embed an existing LRC into a FLAC file (Tab 3)."""
    UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
    flac_dest = UPLOAD_DIR / flac_file.filename
    with open(flac_dest, "wb") as f:
        f.write(await flac_file.read())

    lrc_content = (await lrc_file.read()).decode("utf-8")
    if not lrc_content.strip():
        raise HTTPException(status_code=400, detail="LRC file is empty")

    deezload.embed_lyrics(flac_dest, lrc_content)
    return FileResponse(
        path=str(flac_dest),
        filename=flac_file.filename,
        media_type="audio/flac",
    )


# ── Tab 4: Romanize LRC ──────────────────────────────────────────────

@app.post("/api/lyrics/romanize")
async def romanize_lrc(lrc_file: UploadFile = File(...)):
    """Romanize an LRC file (Tab 4)."""
    content = (await lrc_file.read()).decode("utf-8")
    if not content.strip():
        raise HTTPException(status_code=400, detail="LRC file is empty")

    romaji = deezload.romanize_lyrics(content)
    if not romaji:
        return {"romanized": None, "message": "No CJK characters detected"}

    UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
    out_name = Path(lrc_file.filename).stem + "_romanized.lrc"
    out_path = UPLOAD_DIR / out_name
    out_path.write_text(romaji, encoding="utf-8")

    return {
        "romanized": romaji,
        "download_url": f"/api/files/{out_name}",
        "message": "Romanization complete",
    }


# ── Tab 5: Extract LRC from FLAC ─────────────────────────────────────

@app.post("/api/lyrics/extract")
async def extract_lrc(file: UploadFile = File(...)):
    """Extract embedded lyrics from a FLAC file (Tab 5)."""
    from mutagen.flac import FLAC

    UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
    dest = UPLOAD_DIR / file.filename
    with open(dest, "wb") as f:
        f.write(await file.read())

    audio = FLAC(str(dest))
    lyrics = audio.get("LYRICS", [None])[0]
    if not lyrics:
        raise HTTPException(status_code=404, detail="No embedded lyrics found in this FLAC file")

    is_synced = bool(re.search(r"\[\d{2}:\d{2}\.\d{2,3}\]", lyrics))
    ext = ".lrc" if is_synced else ".txt"
    out_name = Path(file.filename).stem + ext
    out_path = UPLOAD_DIR / out_name
    out_path.write_text(lyrics, encoding="utf-8")

    return {
        "lyrics": lyrics,
        "is_synced": is_synced,
        "download_url": f"/api/files/{out_name}",
    }


# ── Tab 6: Download via Link ─────────────────────────────────────────

@app.post("/api/downloads/link")
async def download_link(link: str = Form(...)):
    """Download from a Spotify/Deezer URL (Tab 6)."""
    tid = _new_task()

    async def _run():
        try:
            client = await get_client()
            _update(tid, "Sending link to bot…", "preparing")
            message = await deezload.send_link_to_bot(client, link, timeout_seconds=15)
            if not message:
                _fail(tid, "No audio reply from bot")
                return

            _update(tid, "Downloading audio…", "downloading")
            file_path = await deezload.download_audio(
                client, message,
                web_progress_callback=lambda r, t: _update_download(tid, r, t),
            )
            if not file_path:
                _fail(tid, "Download failed")
                return

            _update(tid, "Extracting metadata…", "finalizing")
            meta_artist, meta_title = deezload.extract_metadata(file_path)

            _update(tid, "Searching lyrics…")
            lyrics, is_synced = deezload.search_lrclib(meta_artist, meta_title)

            romaji_lyrics = None
            if lyrics:
                _update(tid, "Romanizing lyrics…")
                romaji_lyrics = deezload.romanize_lyrics(lyrics)

            if lyrics and romaji_lyrics:
                _wait_for_choice(tid, file_path, lyrics, romaji_lyrics, is_synced)
            elif lyrics:
                _update(tid, "Embedding lyrics…")
                deezload.embed_lyrics(file_path, lyrics)
                _update(tid, "Saving LRC file…")
                deezload.save_lrc_file(file_path, lyrics, is_synced)
                files = [str(file_path)]
                lrc_path = file_path.with_suffix(".lrc" if is_synced else ".txt")
                if lrc_path.exists():
                    files.append(str(lrc_path))
                _finish(tid, files)
            else:
                _finish(tid, [str(file_path)])
        except Exception as exc:
            _fail(tid, str(exc))

    asyncio.create_task(_run())
    return {"task_id": tid}


# ── File serving ──────────────────────────────────────────────────────

@app.get("/api/files/{filename}")
async def serve_file(filename: str):
    """Serve a file from uploads or downloads directory."""
    for d in (UPLOAD_DIR, DOWNLOAD_DIR):
        path = d / filename
        if path.exists():
            media = "application/octet-stream"
            if filename.endswith(".flac"):
                media = "audio/flac"
            elif filename.endswith(".lrc") or filename.endswith(".txt"):
                media = "text/plain"
            return FileResponse(path=str(path), filename=filename, media_type=media)
    raise HTTPException(status_code=404, detail="File not found")


@app.get("/api/task-files/{task_id}")
async def download_task_files(task_id: str):
    """Get download URLs for completed task files."""
    task = _tasks.get(task_id)
    if not task:
        raise HTTPException(status_code=404, detail="Task not found")
    if not task["done"]:
        raise HTTPException(status_code=400, detail="Task not complete yet")
    if task["error"]:
        raise HTTPException(status_code=500, detail=task["error"])

    file_infos = []
    for fp in task["files"]:
        p = Path(fp)
        if p.exists():
            file_infos.append({
                "filename": p.name,
                "url": f"/api/files/{p.name}",
                "size": p.stat().st_size,
            })
    return {"files": file_infos}


# ── Static file serving (local dev) ──────────────────────────────────
# Mount frontend and assets so the backend can serve everything locally.
PROJECT_ROOT = Path(__file__).resolve().parent.parent

# Assets (images)
if (PROJECT_ROOT / "assets").exists():
    app.mount("/assets", StaticFiles(directory=str(PROJECT_ROOT / "assets")), name="assets")

# Frontend files (HTML, CSS, JS) — must be last (catch-all)
if (PROJECT_ROOT / "lymuru-web").exists():
    app.mount("/", StaticFiles(directory=str(PROJECT_ROOT / "lymuru-web"), html=True), name="frontend")

# Lymuru - DeezLoad Web Dashboard

Lymuru is a web UI and API layer on top of the DeezLoad Telegram bot. It lets you search and download FLAC tracks, embed synced lyrics, and romanize CJK lyrics directly from the browser. The backend reuses the Telegram session created by `deezload.py` so the web UI never needs to handle Telegram authentication.

## Features

- Search and download via `@deezload2bot` inline queries
- Download from Spotify or Deezer links
- Add synced lyrics to existing FLAC files
- Embed LRC into FLAC, extract embedded lyrics, and romanize LRC
- Optional romanized lyrics (Romaji, Pinyin, Korean romanization)
- Task progress with real download percent and file links
- Bulk mode in the UI (search or link lists)

## Requirements

- Python 3.10+
- Telegram API credentials from https://my.telegram.org/apps

## Quick Start (Simple Console Interface)
just double click
 ```bash
   run.bat
   ```

## Quick Start (Local)

1. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

2. Create a `.env` file:
   ```env
   TELEGRAM_API_ID=your_api_id
   TELEGRAM_API_HASH=your_api_hash
   TELEGRAM_PHONE=+628xxxxxxxxxx
   ```

3. Authenticate once to generate the Telegram session file:
   ```bash
   python deezload.py
   ```

4. Run the backend (serves the UI too):
   ```bash
   uvicorn backend.main:app --reload --host 0.0.0.0 --port 8000
   ```

5. Open the app:
   ```
   http://localhost:8000
   ```

## Quick Start (Docker)

1. Copy the env file:
   ```bash
   copy .env.example .env
   ```

2. Fill in Telegram credentials in `.env`.

3. Build and run:
   ```bash
   docker compose up --build
   ```

4. Open the app:
   ```
   http://localhost:3000
   ```

Notes:
- The backend container uses `sessions/`, `downloads/`, and `uploads/` as volumes.
- Run `python deezload.py` once on the host to create `deezload_session.session`. The backend will copy it into `sessions/` on first run.

## Project Structure

```
backend/          FastAPI API server
lymuru-web/       Static UI (HTML/CSS/JS)
assets/           UI images and mascots
downloads/        Downloaded FLAC and LRC output
uploads/          Uploaded files (FLAC/LRC)
sessions/         Telegram session storage
deezload.py       Telegram bot automation core
```

## API Endpoints

- `GET /api/health`
- `GET /api/telegram/status`
- `POST /api/search` (artist, title)
- `POST /api/downloads/choose` (search_key, choice, artist, title)
- `POST /api/downloads/choose-lyrics` (task_id, lyrics_choice)
- `POST /api/lyrics/add` (file, artist, title)
- `POST /api/lyrics/embed` (flac_file, lrc_file)
- `POST /api/lyrics/romanize` (lrc_file)
- `POST /api/lyrics/extract` (file)
- `POST /api/downloads/link` (link)
- `GET /api/downloads/progress/{task_id}`
- `GET /api/task-files/{task_id}`
- `GET /api/files/{filename}`

## Libraries

- Telethon (Telegram API client)
- FastAPI + Uvicorn (backend)
- mutagen (FLAC metadata)
- requests (LRCLIB API)
- python-dotenv (env config)
- pykakasi, pypinyin, korean-romanizer (romanization)

## License

MIT

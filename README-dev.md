# Lymuru - Developer & Contributor Guide

Welcome to the Lymuru developer documentation! This guide will help you understand the project structure, how to set up your local development environment, and the coding standards we follow.

## Local Development Setup 🛠️

1. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

2. **Configure Environment:**
   Create a `.env` file based on `.env.example`:
   ```env
   API_TOKEN=your_api_token_to_login
   TELEGRAM_API_ID=your_api_id
   TELEGRAM_API_HASH=your_api_hash
   TELEGRAM_PHONE=+628xxxxxxxxxx
   ```

3. **Authenticate (First Time Only):**
   Run the bot core directly to generate the Telegram session file (`deezload_session.session`).
   ```bash
   python deezload.py
   ```

4. **Run the Backend (FastAPI):**
   The backend also serves the static UI files.
   ```bash
   uvicorn backend.main:app --reload --host 0.0.0.0 --port 8000
   ```

5. **Access the App:**
   Open `http://localhost:8000` in your browser.

## Project Structure 📁

```text
backend/          # FastAPI API server and routing
lymuru-web/       # Static UI assets (HTML/CSS/JS)
assets/           # UI images and mascots
downloads/        # Downloaded FLAC and LRC output files
uploads/          # User uploaded files for processing (FLAC/LRC)
sessions/         # Telegram session storage
deezload.py       # Core logic: Telegram bot automation (Telethon)
```

## Core Libraries 📚

- **Telethon**: Async Telegram API client used in `deezload.py` to interact with `@deezload2bot`.
- **FastAPI + Uvicorn**: High-performance backend framework.
- **mutagen**: Used for reading and embedding metadata into FLAC audio files.
- **requests**: For querying the LRCLIB API.
- **pykakasi, pypinyin, korean-romanizer**: Tools for parsing and romanizing CJK lyrics.

## API Endpoints 🔌

- `GET /api/health` - Server health check
- `GET /api/telegram/status` - Check Telegram session status
- `POST /api/search` - Search tracks (artist, title)
- `POST /api/downloads/choose` - Initiate download for a specific track
- `POST /api/downloads/choose-lyrics` - Select and download lyrics
- `POST /api/lyrics/add` - Append lyrics to an existing track
- `POST /api/lyrics/embed` - Embed LRC data into a FLAC file
- `POST /api/lyrics/romanize` - Romanize a given LRC file
- `POST /api/lyrics/extract` - Extract LRC from a FLAC file
- `POST /api/downloads/link` - Download via direct Spotify/Deezer link
- `GET /api/downloads/progress/{task_id}` - Poll for task status
- `GET /api/task-files/{task_id}` - Get resulting files from a task
- `GET /api/files/{filename}` - Serve static/downloaded files

## Coding Standards & Contribution Guidelines 🤝

1. **Keep it Simple**: Write clear and explicit code. Avoid unnecessary abstractions.
2. **Comment Generously**: Ensure complex logic, especially within `deezload.py` and the FastAPI routers, is well-documented with inline comments and docstrings. We want this codebase to be accessible to junior developers.
3. **Type Hinting**: Use Python type hints in function signatures to make inputs and outputs predictable.
4. **Error Handling**: Do not silently swallow errors. Fail gracefully and return clear HTTP error codes via FastAPI.
5. **License**: All contributions fall under the [MIT License](LICENSE). 

By contributing, you agree to release your code under the MIT License. Happy coding!

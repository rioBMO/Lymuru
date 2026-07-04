# Lymuru - DeezLoad Web Dashboard

![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Python 3.10+](https://img.shields.io/badge/python-3.10+-blue.svg)

Lymuru is a beautiful web UI and API layer on top of the DeezLoad Telegram bot. It lets you search and download FLAC tracks, embed synced lyrics, and romanize CJK lyrics directly from the browser. It abstracts away the complexity of interacting with the Telegram bot directly.

> 🛠 **Are you a Developer?** Please check our [README-dev.md](README-dev.md) for architecture details, API endpoints, and contribution guidelines.

## Features ✨

- **Search & Download**: Find and download high-quality FLAC audio via `@deezload2bot` inline queries.
- **Link Support**: Direct downloads from Spotify or Deezer links.
- **Lyrics Integration**: 
  - Add synced lyrics to existing FLAC files.
  - Embed LRC into FLAC, extract embedded lyrics, and romanize LRC.
- **Romanization**: Optional romanized lyrics for CJK languages (Romaji, Pinyin, Korean romanization).
- **Bulk Mode**: Download multiple tracks efficiently.
- **Real-time Progress**: Monitor your tasks with live download percentages.

## Requirements 📋

- Python 3.10+
- Telegram API credentials (API ID and Hash) from https://my.telegram.org/apps
- A valid Telegram Account

## Quick Start (Simple Windows Console) 🚀

If you are on Windows, simply double-click the run script:
```cmd
run.bat
```
*(Make sure you have filled in your credentials in `.env` if prompted)*

## Quick Start (Docker) 🐳

The easiest way to run the application in an isolated environment.

1. **Copy the environment file:**
   ```bash
   copy .env.example .env
   ```

2. **Configure your credentials:**
   Fill in your Telegram credentials (`TELEGRAM_API_ID`, `TELEGRAM_API_HASH`, etc.) in the `.env` file.

3. **Build and start the container:**
   ```bash
   docker compose up --build
   ```

4. **Open your browser:**
   Navigate to http://localhost:3000

*Note: You may need to run `python deezload.py` once on your host machine to generate the initial `deezload_session.session` file before running Docker.*

## License 📄

This project is licensed under the [MIT License](LICENSE).

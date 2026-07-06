# Lymuru

![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Go 1.25+](https://img.shields.io/badge/Go-1.25+-blue.svg)
![Wails v2](https://img.shields.io/badge/Wails-v2-blueviolet)

Lymuru is a native desktop application for searching and downloading high-quality music (FLAC, MP3, M4A) from multiple providers. Built with Go, Wails v2, and React, it combines a rich search experience with a full suite of audio tools — all running locally on your machine.

> 🛠 **Developer?** See [README-dev.md](README-dev.md) for architecture details, setup instructions, and contribution guidelines.

---

## Features

- **Multi-source downloads** — Search and download from Tidal, Amazon Music, Qobuz, and (optionally) Deezer via a Python sidecar.
- **Smart fallback** — Configure a provider priority order; Lymuru automatically falls back when a track isn't found on your first-choice provider.
- **Download queue & history** — Real-time progress tracking with a queue view, plus persistent history backed by SQLite.
- **Lyrics tools** — Fetch synced lyrics from LRCLIB, add lyrics to existing tracks, embed/extract LRC files, and romanize CJK lyrics (Romaji, Pinyin, Korean romanization).
- **Audio tools** — Converter (FLAC ↔ MP3 ↔ M4A), spectrum analysis, and sample-rate resampler.
- **File Manager** — Browse, open, and delete downloaded tracks in your configured downloads folder.
- **API Status** — Check connectivity and health for all configured services.
- **Auto-download FFmpeg** — No manual dependency setup. Lymuru manages FFmpeg transparently on first launch.
- **Themes** — Light and dark modes, featuring Lymuru's own cyan/blue palette.
- **Drag & drop** — Drop track links directly onto the window.

---

## Requirements

- **Windows 10 / 11** (primary platform)
- No external dependencies for core features — just run the executable
- **Optional:** Python 3.11+ if you enable the Deezer sidecar

---

## Quick Start

1. Download the latest release from the [Releases](https://github.com/lymuru/lymuru/releases) page.
2. Run `Lymuru.exe`.
3. The app automatically downloads and configures FFmpeg on first launch.
4. Start searching and downloading!

### Deezer Sidecar (optional)

If you want Deezer as an additional download source:

1. Open **Settings → Deezer**.
2. Enable the sidecar and set the path to your Python 3.11+ executable.
3. Enter your Telegram API ID, API Hash, and phone number (**Settings → Deezer → Set Credentials**).
4. When prompted, enter the Telegram verification code in the auth dialog.
5. Your session is saved to `data/deezload_session.session` and reused on subsequent launches.

---

## Development

```bash
# Prerequisites: Go 1.25+, Bun, Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install frontend dependencies
cd Lymuru/frontend
bun install

# Start dev server (hot reload for Go + React)
cd Lymuru
wails dev

# Build native executable
wails build
# Output: build/bin/Lymuru.exe
```

Full development guide: [README-dev.md](README-dev.md).

---

## Project Structure

```
Lymuru/
├── main.go                  # Wails entrypoint
├── app.go                   # App struct and Wails bindings
├── app_downloads.go         # Download orchestration bindings
├── wails.json               # Wails config
├── backend/                 # Go packages
│   ├── tidal.go             # Tidal provider
│   ├── qobuz.go / qobuz_api.go  # Qobuz provider
│   ├── amazon.go            # Amazon Music provider
│   ├── deezer_sidecar.go    # Python sidecar manager
│   ├── spotfetch.go         # Spotify metadata scraper
│   ├── lyrics.go            # LRCLIB integration
│   ├── history.go           # Download history (SQLite)
│   ├── config.go            # User settings
│   ├── ffmpeg.go            # FFmpeg discovery / auto-download
│   ├── resample.go          # Audio resampler
│   ├── analysis.go          # Audio analysis
│   ├── filemanager.go       # File browsing
│   ├── logger.go            # Structured file logger
│   ├── keychain.go          # OS keychain for credentials
│   ├── progress.go          # In-memory download queue
│   └── storage/             # SQLite database layer
├── frontend/                # React + TypeScript + Vite + Bun
│   └── src/
│       ├── components/      # UI components (pages, dialogs, primitives)
│       ├── hooks/           # React hooks
│       └── lib/             # Utilities and settings
├── sidecar/
│   └── deezload.py          # Python sidecar (Deezer via Telegram bot)
└── data/                    # Runtime data (created at first launch)
    ├── lymuru.db            # SQLite database (history, settings)
    ├── logs/lymuru.log      # Application logs
    └── deezload_session.session  # Telegram session (if sidecar is used)
```

---

## License

This project is licensed under the [MIT License](LICENSE).

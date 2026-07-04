# Lymuru

![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Go 1.21+](https://img.shields.io/badge/Go-1.21+-blue.svg)

Lymuru is a beautiful desktop application for searching and downloading high-quality FLAC/MP3 audio. It also features integrated lyrics management, letting you fetch synced lyrics, embed LRC into FLAC files, extract embedded lyrics, and romanize CJK lyrics. 

Lymuru runs purely natively on your machine, leveraging Go, Wails v2, and React for a blazingly fast and premium user experience.

> 🛠 **Are you a Developer?** Please check our [README-dev.md](README-dev.md) for architecture details, API endpoints, and contribution guidelines.

## Features ✨

- **Search & Download**: Find and download high-quality FLAC/MP3/M4A audio natively.
- **Auto-Download FFmpeg**: No manual dependencies needed. Lymuru manages FFmpeg transparently.
- **Lyrics Integration**: 
  - Fetch synced lyrics from LRCLIB and embed them into existing FLAC files.
  - Embed local LRC into FLAC, extract embedded lyrics, and romanize LRC.
- **Romanization**: Optional romanized lyrics for CJK languages (Romaji, Pinyin, Korean romanization).
- **History & Queue**: A fully featured queue and download history manager, backed by SQLite.

## Requirements 📋

- **Windows 10/11** (macOS and Linux support incoming)
- No external dependencies needed! Just run the executable.

## Quick Start 🚀

1. Download the latest release from the Releases page.
2. Run `Lymuru.exe`.
3. The app will automatically configure FFmpeg on first launch.

*Note: For developers, run `wails dev` to start the live development server, or `wails build` to compile the native executable.*

## License 📄

This project is licensed under the [MIT License](LICENSE).

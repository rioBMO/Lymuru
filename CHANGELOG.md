# Changelog

All notable changes to Lymuru will be documented in this file.

## [Unreleased] - 2026-07-04
### Added
- **Native Desktop App**: Lymuru is now a native desktop application powered by Go and Wails v2, running locally on Windows.
- **Embedded React UI**: The React UI has been entirely redesigned, inheriting the sleek SpotiFLAC visual language but retaining Lymuru's classic cyan & blue styling.
- **Go Backend Download Engine**: Downloads are now handled directly by the Go backend (via Tidal/Amazon/Qobuz source integration), eliminating the need for a separate Python sidecar or Telegram bots.
- **SQLite Database**: Replaced JSON file storage with a robust modernc.org SQLite DB for Download History and Application Settings.
- **FFmpeg Auto-Downloader**: No more manual installations! The app automatically downloads and verifies FFmpeg binaries on first launch.
- **Lyrics & Romanization tools**: Retained and optimized the Lyrics Manager tools for Add, Embed, Extract, and Romanize (CJK) operations directly via the Go backend.

### Removed
- The `deezload.py` Python sidecar has been completely deprecated and removed.
- Telegram Auth Dialogs and bot integration flows have been removed.
- Python FastAPI endpoints and HTTP polling have been replaced with fast, native Wails event bindings.

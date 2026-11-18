# Changelog

All notable changes to Lymuru will be documented in this file.

## [1.2.0] - 2026-11-19

This release makes the buttons that did nothing work, or removes them.

### Changed
- **Playlist folders**: with **Create playlist folder** on (the default), each playlist now gets its own folder named after the playlist, or `<playlist> - <owner>` when **Playlist Owner Folder Name** is on. Before, every playlist went into one folder called `playlist`. New downloads go to the new folders; files you already downloaded aren't moved.
- **Open folder** shows an error if the folder doesn't exist, instead of quietly opening a different one.

### Fixed
- **Cover and artist image downloads**: the per-track cover button, **Download All Covers**, and the album, playlist, header, avatar and gallery image buttons failed on every click. They now save the image next to the track's audio, or in the artist's folder, with progress, a Stop button and a summary.
- **Lyrics downloads**: the per-track lyrics button and **Download All Lyrics** now save an `.lrc` file next to the track's audio. Synced lyrics are used when available, then plain lyrics. Tracks with no lyrics are marked as skipped.
- **Lyrics Manager**: **Add from LRCLIB**, **Embed .lrc** and **Romanize** can be opened from Lyrics Manager again. They accept FLAC, MP3 and M4A, and show a message for each result. If you leave artist and title empty, **Add from LRCLIB** uses the file's own tags.
- **Open folder** buttons did nothing. They now open the folder in Explorer. **Open in folder** in Romanize opens the folder that contains the `.lrc` file.
- Clicking an artist name could crash. Each artist in a multi-artist track is now its own link, and an artist that can't be found shows "Artist not found".
- **Play Preview** in History did nothing. It plays and stops again, and shows a message when a track has no preview.
- The title-bar **Website** item now opens the Lymuru GitHub page.

### Removed
- The title-bar **Network** section (IP address, country and "Blocked by Spotify"). It never worked, and the app no longer looks up your IP address.
- The **Check availability** buttons, which never worked.

## [1.1.1] - 2026-11-18

### Fixed
- The app header and update dialog showed "v1.0.0" in the 1.1.0 release. The UI now reads its version from `wails.json`, the same source as the backend and the release workflow.
- The update check compares versions numerically, so 1.10.0 is correctly treated as newer than 1.9.0.

## [1.1.0] - 2026-11-18

First version published as a GitHub Release, with a Windows installer, a portable EXE and a ZIP.

### Added
- **Windows installer**: an NSIS setup that installs to Program Files and adds Start Menu and Desktop shortcuts plus an uninstaller. It installs WebView2 if missing.
- **Optional Deezer source, bundled**: the Telegram-based Deezer sidecar ships inside the app again and is unpacked when needed. It requires Python 3.11+ and your own Telegram API credentials (Settings → Deezer).
- Clear startup errors: if Lymuru can't start, it shows an error dialog instead of closing silently.

### Changed
- **Data location**: the database, logs and Telegram session now live in `%APPDATA%\Lymuru` instead of a `data` folder next to the app. An existing `data\lymuru.db` is copied over automatically on first launch.
- Deezer settings apply immediately. Saving credentials also enables Deezer, and turning it on or off starts or stops it without restarting the app.
- Python is found in its standard install folder even when it isn't on PATH or was installed while the app was open.
- Updated Wails to v2.17.0.

### Fixed
- The installed app (Program Files) failed to start because it tried to write its data next to the executable.
- A possible crash when stopping the Deezer sidecar.

## [1.0.1] – [1.0.3] - 2026-07-06 – 2026-07-10

Build and release pipeline fixes only; no app changes. No GitHub Release was published for these tags.

## [1.0.0] - 2026-07-06
### Added
- **Native Desktop App**: Lymuru is now a native desktop application powered by Go and Wails v2, running locally on Windows.
- **Embedded React UI**: The React UI has been entirely redesigned, inheriting the sleek SpotiFLAC visual language but retaining Lymuru's classic cyan & blue styling.
- **Go Backend Download Engine**: Downloads are now handled directly by the Go backend (via Tidal/Amazon/Qobuz source integration), eliminating the need for a separate Python sidecar or Telegram bots.
- **SQLite Database**: Replaced JSON file storage with a robust modernc.org SQLite DB for Download History and Application Settings.
- **FFmpeg Auto-Downloader**: No more manual installations! The app automatically downloads and verifies FFmpeg binaries on first launch.
- **Lyrics & Romanization tools**: Retained and optimized the Lyrics Manager tools for Add, Embed, Extract, and Romanize (CJK) operations directly via the Go backend.

### Removed
- The `deezload.py` Python sidecar was deprecated as a required component. It returned as an optional, bundled source in 1.1.0.
- Telegram Auth Dialogs and bot integration flows have been removed.
- Python FastAPI endpoints and HTTP polling have been replaced with fast, native Wails event bindings.

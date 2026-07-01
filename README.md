# DeezLoad — Telegram Music Downloader + Lyrics

A comprehensive Python CLI tool that automates music downloading and lyrics management. Features multiple modes for downloading via **DeezLoad** Telegram bot, direct **Spotify/Deezer** links, and advanced lyrics processing with **romanization** support for Japanese, Chinese, and Korean.

## Features

- **6 operation modes** — Download, embed, extract, and romanize lyrics
- **Bot search mode** — Search and download via `@deezload2bot` inline queries
- **Direct link mode** — Download from Spotify/Deezer URLs sent to bot
- **Local file mode** — Add lyrics to existing FLAC files
- **LRC management** — Embed, extract, and romanize `.lrc` files
- **Synced lyrics** — Fetches timestamped lyrics from [LRCLIB](https://lrclib.net)
- **CJK romanization** — Auto-converts Japanese (romaji), Chinese (pinyin), Korean lyrics
- **FLAC embedding** — Writes lyrics into FLAC Vorbis comment tags
- **Progress tracking** — Real-time download progress with speed indicator

## Requirements

- Python 3.10+
- Telegram API credentials ([my.telegram.org/apps](https://my.telegram.org/apps))

## Setup

1. Clone the repo:
   ```bash
   git clone https://github.com/YOUR_USERNAME/deezload.git
   cd deezload
   ```

2. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

3. **Optional romanization dependencies** (for CJK lyrics):
   ```bash
   pip install pykakasi pypinyin korean-romanizer
   ```

4. Create a `.env` file:
   ```env
   TELEGRAM_API_ID=your_api_id
   TELEGRAM_API_HASH=your_api_hash
   TELEGRAM_PHONE=+628xxxxxxxxxx
   ```

5. Run:
   ```bash
   python deezload.py
   ```

## Usage

The application offers 6 modes of operation:

| Mode | Purpose |
|------|---------|
| **1** | Search and download music via DeezLoad bot inline queries |
| **2** | Add lyrics to existing FLAC files on your computer |
| **3** | Embed existing LRC files into FLAC metadata |
| **4** | Romanize existing LRC files for CJK lyrics |
| **5** | Extract embedded lyrics from FLAC files to LRC |
| **6** | Download music using direct Spotify/Deezer links |

### Mode 1: Search & Download from Bot
```
Select mode [1-6]: 1
Artist: Kenshi Yonezu
Title:  KAZE

[SEARCH] Sending inline query to @deezload2bot: KAZE Kenshi Yonezu
[OK] Got 5 result(s)

──────────────────────────────────────────────────
  [1] KAZE - Kenshi Yonezu (FLAC)
  [2] KAZE - Kenshi Yonezu (MP3 320)
──────────────────────────────────────────────────

Select track [1-2]: 1
...
  [1] Original lyrics
  [2] Romanized lyrics
LRC format [1-2]: 2

[OK] Lyrics embedded
[OK] Saved Kenshi Yonezu - KAZE.lrc
[DONE] File ready: downloads\Kenshi Yonezu - KAZE.flac
```

### Mode 2: Add Lyrics to Existing FLAC
```
Select mode [1-6]: 2
FLAC file path: C:\Music\Wonstein - Promise.flac
Artist [Wonstein]:
Title  [Promise]:

[OK] Synced lyrics found
[ROMAJI] Detected language: Korean
[ROMAJI] Converting lyrics...
[OK] Lyrics embedded
[OK] Saved Wonstein - Promise.lrc
```

### Mode 3: Embed Existing LRC into FLAC
```
Select mode [1-6]: 3
FLAC file path: C:\Music\song.flac
LRC file path:  C:\Music\song.lrc

[OK] Lyrics embedded
[DONE] Lyrics embedded into: C:\Music\song.flac
```

### Mode 4: Romanize Existing LRC File
```
Select mode [1-6]: 4
LRC file path: C:\Music\japanese_song.lrc

[ROMAJI] Detected language: Japanese
[ROMAJI] Converting lyrics...
[DONE] Romanized LRC saved: C:\Music\japanese_song.lrc
```

### Mode 5: Extract LRC from FLAC File
```
Select mode [1-6]: 5
FLAC file path: C:\Music\song_with_lyrics.flac

[OK] Extracted synced lyrics to: song_with_lyrics.lrc
[DONE] Lyrics saved: C:\Music\song_with_lyrics.lrc
```

### Mode 6: Download from Spotify/Deezer Link
```
Select mode [1-6]: 6
Spotify/Deezer link: https://open.spotify.com/track/...

[SEND] Sending link to @deezload2bot: https://open.spotify.com/track/...
[OK] Audio message received from bot
[DOWNLOAD] Downloading: Artist - Song.flac (12.3 MB)
[████████████████████████████████] 100.0%  12.3/12.3 MB  1250 KB/s
[OK] Synced lyrics found
[DONE] File ready: downloads\Artist - Song.flac
```

## Libraries

| Library | Purpose |
|---------|---------|
| [Telethon](https://github.com/LonamiWebs/Telethon) | Telegram client API for bot interaction |
| [mutagen](https://github.com/quodlibet/mutagen) | FLAC metadata reading and editing |
| [requests](https://github.com/psf/requests) | LRCLIB API requests |
| [python-dotenv](https://github.com/theskumar/python-dotenv) | Environment variable management |
| [pykakasi](https://github.com/miurahr/pykakasi) | Japanese → Romaji conversion |
| [pypinyin](https://github.com/mozillazg/python-pinyin) | Chinese → Pinyin conversion |
| [korean-romanizer](https://github.com/osori/korean-romanizer) | Korean → Romanization |

## License

MIT

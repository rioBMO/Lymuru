"""
DeezLoad Telegram Bot Automation Tool

Automates song downloading via the DeezLoad Telegram bot,
extracts metadata, fetches synced lyrics from LRCLIB,
and embeds them into FLAC files.
"""

import asyncio
import os
import re
import sys
import unicodedata
from pathlib import Path
from typing import Optional, Callable

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
DOWNLOAD_DIR: Path = Path("downloads")
LRCLIB_BASE: str = "https://lrclib.net/api/get"
RESULT_TIMEOUT: int = 120  # seconds to wait for bot response


async def connect_telegram() -> TelegramClient:
    """Initialize and connect a Telegram client using stored credentials.

    Returns:
        An authenticated TelegramClient instance.

    Raises:
        SystemExit: If API_ID or API_HASH are missing.
    """
    if not API_ID or not API_HASH:
        print("[ERROR] TELEGRAM_API_ID and TELEGRAM_API_HASH must be set in .env")
        sys.exit(1)

    client = TelegramClient(SESSION_NAME, API_ID, API_HASH)
    await client.start(phone=PHONE)
    print("[OK] Connected to Telegram")
    return client


async def search_song(client: TelegramClient, artist: str, title: str) -> Optional[InlineResults]:
    """Send an inline query to the DeezLoad bot and return results.

    Args:
        client: Authenticated TelegramClient.
        artist: Artist name.
        title: Song title.

    Returns:
        InlineResults from the bot, or None if no results.
    """
    query = f"{title} {artist}"
    print(f"[SEARCH] Sending inline query to @{BOT_USERNAME}: {query}")

    results = await client.inline_query(BOT_USERNAME, query)

    if not results:
        print("[WARN] No results returned by bot.")
        return None

    print(f"[OK] Got {len(results)} result(s)")
    return results


def display_results(results: InlineResults) -> None:
    """Display inline query results for user selection.

    Args:
        results: InlineResults from the inline query.
    """
    print()
    print("─" * 50)
    for i, result in enumerate(results):
        title = getattr(result, 'title', None) or '(no title)'
        description = getattr(result, 'description', None) or ''
        print(f"  [{i + 1}] {title}")
        if description:
            print(f"      {description}")
    print("─" * 50)


async def select_and_send_result(
    client: TelegramClient,
    results: InlineResults,
    choice: int,
) -> Optional[object]:
    """Click an inline result to send it to Saved Messages, then retrieve the message.

    Args:
        client: Authenticated TelegramClient.
        results: InlineResults from the inline query.
        choice: 0-based index of the selected result.

    Returns:
        The Telethon message containing the audio, or None on failure.
    """
    me = await client.get_me()

    # Record the latest message ID BEFORE clicking, so we only look at newer messages
    recent = await client.get_messages("me", limit=1)
    last_id = recent[0].id if recent else 0

    print(f"[SELECT] Sending result #{choice + 1} to Saved Messages...")

    try:
        await results[choice].click(entity=me.id)
    except Exception as exc:
        print(f"[ERROR] Failed to send inline result: {exc}")
        return None

    # Poll Saved Messages for NEW messages only (after last_id)
    print("[WAIT] Waiting for audio in Saved Messages...")
    for attempt in range(RESULT_TIMEOUT // 2):
        await asyncio.sleep(2)
        messages = await client.get_messages("me", limit=5, min_id=last_id)
        for msg in messages:
            if msg.document:
                for attr in msg.document.attributes:
                    if isinstance(attr, (DocumentAttributeAudio, DocumentAttributeFilename)):
                        print("[OK] Audio message found in Saved Messages")
                        return msg
            if msg.audio:
                print("[OK] Audio message found in Saved Messages")
                return msg

    print("[TIMEOUT] No audio appeared in Saved Messages.")
    return None


async def download_audio(
    client: TelegramClient,
    message: object,
    web_progress_callback: Optional[Callable[[int, int], None]] = None
) -> Optional[Path]:
    """Download the audio file from a Telegram message.

    If multiple formats are available, FLAC is preferred.

    Args:
        client: Authenticated TelegramClient.
        message: The Telethon message object containing the audio document.
        web_progress_callback: Optional callback receiving (received_bytes, total_bytes).

    Returns:
        Path to the downloaded file, or None on failure.
    """
    DOWNLOAD_DIR.mkdir(parents=True, exist_ok=True)

    # Determine filename from document attributes
    filename = "unknown_track"
    for attr in message.document.attributes:
        if isinstance(attr, DocumentAttributeFilename):
            filename = attr.file_name
            break
        if isinstance(attr, DocumentAttributeAudio):
            performer = attr.performer or "Unknown"
            attr_title = attr.title or "Unknown"
            filename = f"{performer} - {attr_title}"

    # Sanitize filename
    filename = re.sub(r'[<>:"/\\|?*]', '_', filename)

    # Ensure it has an extension
    if not Path(filename).suffix:
        filename += ".flac"

    import time

    dest = DOWNLOAD_DIR / filename
    file_size = message.document.size or 0
    size_mb = file_size / (1024 * 1024)
    print(f"[DOWNLOAD] Downloading: {filename} ({size_mb:.1f} MB)")

    start_time = time.time()

    def progress_callback(received: int, total: int) -> None:
        t = total or file_size or 1
        pct = received / t * 100
        bar_len = 30
        filled = int(bar_len * received // t)
        bar = "█" * filled + "░" * (bar_len - filled)
        elapsed = time.time() - start_time
        speed = received / elapsed / 1024 if elapsed > 0 else 0
        print(f"\r  [{bar}] {pct:5.1f}%  {received / 1048576:.1f}/{t / 1048576:.1f} MB  {speed:.0f} KB/s", end="", flush=True)
        if web_progress_callback:
            web_progress_callback(received, t)

    # Use download_file with larger part_size (512KB) for faster throughput
    with open(dest, "wb") as f:
        await client.download_file(
            message.document,
            f,
            part_size_kb=512,
            progress_callback=progress_callback,
        )
    print()  # newline after progress bar

    elapsed = time.time() - start_time
    if dest.exists() and dest.stat().st_size > 0:
        print(f"[OK] Saved to {dest} ({elapsed:.1f}s)")
        return dest

    print("[ERROR] Download failed")
    return None


def search_lrclib(artist: str, title: str) -> tuple[Optional[str], bool]:
    """Query LRCLIB API for synced lyrics, falling back to plain lyrics.

    Args:
        artist: Artist name from file metadata.
        title: Track title from file metadata.

    Returns:
        A (lyrics, is_synced) tuple. lyrics is None if nothing is available.
    """
    print(f"[LYRICS] Searching LRCLIB for: {artist} - {title}")

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
            print("[OK] Synced lyrics found")
            return synced, True
        plain = data.get("plainLyrics")
        if plain:
            print("[OK] Plain lyrics found (no synced lyrics available)")
            return plain, False
        print("[WARN] No lyrics available for this track")
        return None, False
    except requests.RequestException as exc:
        print(f"[ERROR] LRCLIB request failed: {exc}")
        return None, False


def detect_language(text: str) -> Optional[str]:
    """Detect if text contains CJK characters and return the dominant language.

    Args:
        text: Text to analyze (lyrics without timestamps).

    Returns:
        'ja' for Japanese, 'zh' for Chinese, 'ko' for Korean, or None.
    """
    # Strip LRC timestamps like [00:12.34]
    clean = re.sub(r"\[\d{2}:\d{2}\.\d{2,3}\]", "", text)

    ja_count = 0  # Hiragana + Katakana
    zh_count = 0  # CJK Unified Ideographs (without kana = Chinese)
    ko_count = 0  # Hangul

    for ch in clean:
        name = unicodedata.name(ch, "")
        if "HIRAGANA" in name or "KATAKANA" in name:
            ja_count += 1
        elif "CJK" in name:
            zh_count += 1
        elif "HANGUL" in name:
            ko_count += 1

    # Japanese uses kanji (CJK) + kana, so if kana present → Japanese
    if ja_count > 0:
        return "ja"
    if ko_count > 0:
        return "ko"
    if zh_count > 5:
        return "zh"
    return None


def detect_line_languages(text: str) -> set[str]:
    """Detect all CJK languages present in a text.

    Args:
        text: Text to analyze.

    Returns:
        Set of language codes found ('ja', 'zh', 'ko').
    """
    langs = set()
    for ch in text:
        name = unicodedata.name(ch, "")
        if "HIRAGANA" in name or "KATAKANA" in name:
            langs.add("ja")
        elif "HANGUL" in name:
            langs.add("ko")
        elif "CJK" in name:
            langs.add("zh")
    # CJK ideographs with kana = Japanese kanji, not Chinese
    if "ja" in langs and "zh" in langs:
        langs.discard("zh")
    return langs


def romanize_lyrics(lyrics: str) -> Optional[str]:
    """Convert synced lyrics to romanized form (romaji/pinyin/romanization).

    Handles multilingual lyrics by detecting the language of each line
    and applying the appropriate romanizer. Preserves LRC timestamps.

    Args:
        lyrics: Synced lyrics string with LRC timestamps.

    Returns:
        Romanized lyrics string, or None if no CJK detected.
    """
    # Check what languages are present across all lyrics
    all_langs = detect_line_languages(re.sub(r"\[\d{2}:\d{2}\.\d{2,3}\]", "", lyrics))
    if not all_langs:
        return None

    lang_names = {"ja": "Japanese", "zh": "Chinese", "ko": "Korean"}
    detected = ", ".join(lang_names.get(l, l) for l in sorted(all_langs))
    print(f"[ROMAJI] Detected language(s): {detected}")

    # Load converters for each detected language
    converters: dict[str, object] = {}

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
        """Pick the right converter based on the characters in this line."""
        line_langs = detect_line_languages(text)
        # Try each detected language in order of priority
        for lang in ("ja", "ko", "zh"):
            if lang in line_langs and lang in converters:
                return converters[lang](text)
        # No CJK detected on this line — return as-is
        return text

    print("[ROMAJI] Converting lyrics...")
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

    print("[OK] Romanization complete")
    return "\n".join(output_lines)


def embed_lyrics(file_path: Path, lyrics: str) -> None:
    """Embed synced lyrics into a FLAC file's Vorbis comment LYRICS tag.

    Args:
        file_path: Path to the FLAC file.
        lyrics: The synced lyrics string to embed.
    """
    if file_path.suffix.lower() != ".flac":
        print(f"[SKIP] Lyrics embedding only supported for FLAC — got {file_path.suffix}")
        return

    print(f"[EMBED] Embedding lyrics into {file_path.name}")
    audio = FLAC(str(file_path))
    audio["LYRICS"] = lyrics
    audio.save()
    print("[OK] Lyrics embedded")


def choose_lyrics_format(lyrics: str, romaji_lyrics: Optional[str]) -> str:
    """Ask the user which lyrics format to use for the .lrc file.

    Args:
        lyrics: Original synced lyrics.
        romaji_lyrics: Romanized lyrics, or None if not CJK.

    Returns:
        The chosen lyrics string.
    """
    if not romaji_lyrics:
        return lyrics

    print()
    print("  [1] Original lyrics")
    print("  [2] Romanized lyrics")
    while True:
        try:
            choice = int(input("LRC format [1-2]: ").strip())
            if choice == 1:
                return lyrics
            if choice == 2:
                return romaji_lyrics
            print("  Enter 1 or 2")
        except ValueError:
            print("  Enter a valid number")


def save_lrc_file(file_path: Path, lyrics: str, is_synced: bool = True) -> None:
    """Save a lyrics file alongside the audio file.

    Synced lyrics are saved as .lrc; plain lyrics as .txt.

    Args:
        file_path: Path to the audio file (used to derive lyrics filename).
        lyrics: Lyrics string to write.
        is_synced: Whether the lyrics are synced (timestamped).
    """
    ext = ".lrc" if is_synced else ".txt"
    out_path = file_path.with_suffix(ext)
    out_path.write_text(lyrics, encoding="utf-8")
    print(f"[OK] Saved {out_path.name}")


def extract_metadata(file_path: Path) -> tuple[str, str]:
    """Extract artist and title metadata from a FLAC file.

    Falls back to filename parsing if tags are missing.

    Args:
        file_path: Path to the audio file.

    Returns:
        A (artist, title) tuple.
    """
    if file_path.suffix.lower() == ".flac":
        try:
            audio = FLAC(str(file_path))
            artist = audio.get("artist", [None])[0]
            title = audio.get("title", [None])[0]
            if artist and title:
                print(f"[META] Metadata: {artist} — {title}")
                return artist, title
        except Exception as exc:
            print(f"[WARN] Could not read FLAC tags: {exc}")

    # Fallback: parse "Artist - Title.ext" from filename
    stem = file_path.stem
    if " - " in stem:
        parts = stem.split(" - ", 1)
        return parts[0].strip(), parts[1].strip()
    return "Unknown", stem


async def send_link_to_bot(
    client: TelegramClient, link: str,
) -> Optional[object]:
    """Send a Spotify/Deezer link directly to the DeezLoad bot and wait for an audio reply.

    The bot typically replies with an info message first, then the audio file.
    This function waits until a message with a downloadable audio document appears.

    Args:
        client: Authenticated TelegramClient.
        link: A Spotify or Deezer URL.

    Returns:
        The Telethon message containing the audio document, or None on timeout.
    """
    bot = await client.get_entity(BOT_USERNAME)
    print(f"[SEND] Sending link to @{BOT_USERNAME}: {link}")
    sent = await client.send_message(bot, link)
    sent_id = sent.id

    print("[WAIT] Waiting for audio reply from bot...")
    for _ in range(RESULT_TIMEOUT // 2):
        await asyncio.sleep(2)
        messages = await client.get_messages(bot, limit=5, min_id=sent_id)
        for msg in messages:
            if not msg.out and msg.document:
                for attr in msg.document.attributes:
                    if isinstance(attr, (DocumentAttributeAudio, DocumentAttributeFilename)):
                        print("[OK] Audio message received from bot")
                        return msg
            if not msg.out and msg.audio:
                print("[OK] Audio message received from bot")
                return msg

    print("[TIMEOUT] No audio reply from bot.")
    return None


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
        # --- Extract LRC from FLAC ---
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
        print()
        print(f"[DONE] Lyrics saved: {out_path}")
        return

    if mode == 3:
        # --- Embed existing LRC into FLAC ---
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
        print()
        print(f"[DONE] Lyrics embedded into: {flac_path}")
        return

    if mode == 4:
        # --- Romanize existing LRC file ---
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
        print()
        print(f"[DONE] Romanized LRC saved: {lrc_path}")
        return

    if mode == 6:
        # --- Spotify/Deezer link mode ---
        link = input("Spotify/Deezer link: ").strip()
        if not link:
            print("[ERROR] A link is required.")
            sys.exit(1)

        client = await connect_telegram()
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
                chosen = choose_lyrics_format(lyrics, romaji_lyrics)
                embed_lyrics(file_path, chosen)
                save_lrc_file(file_path, chosen, is_synced)
            else:
                print("[DONE] No lyrics to embed.")

            print()
            print(f"[DONE] File ready: {file_path}")
        finally:
            await client.disconnect()
            print("[OK] Disconnected from Telegram")
        return

    if mode == 2:
        # --- Local file mode ---
        file_input = input("FLAC file path: ").strip().strip('"')
        file_path = Path(file_input)
        if not file_path.exists():
            print(f"[ERROR] File not found: {file_path}")
            sys.exit(1)
        if file_path.suffix.lower() != ".flac":
            print(f"[ERROR] Expected .flac file, got {file_path.suffix}")
            sys.exit(1)

        meta_artist, meta_title = extract_metadata(file_path)

        # Allow manual override
        artist_input = input(f"Artist [{meta_artist}]: ").strip() or meta_artist
        title_input = input(f"Title  [{meta_title}]: ").strip() or meta_title

        lyrics, is_synced = search_lrclib(artist_input, title_input)

        romaji_lyrics = None
        if lyrics:
            romaji_lyrics = romanize_lyrics(lyrics)
            chosen = choose_lyrics_format(lyrics, romaji_lyrics)
            embed_lyrics(file_path, chosen)
            save_lrc_file(file_path, chosen, is_synced)
        else:
            print("[DONE] No lyrics found.")

        print()
        print(f"[DONE] File ready: {file_path}")
        return

    # --- DeezLoad bot mode ---
    artist_input = input("Artist: ").strip()
    title_input = input("Title:  ").strip()

    if not artist_input or not title_input:
        print("[ERROR] Artist and title are required.")
        sys.exit(1)

    client = await connect_telegram()

    try:
        # Step 1: Inline query
        results = await search_song(client, artist_input, title_input)
        if not results:
            print("[ABORT] No results found. Exiting.")
            return

        # Step 2: Let user pick a result
        display_results(results)
        while True:
            try:
                pick = int(input(f"\nSelect track [1-{len(results)}]: ").strip())
                if 1 <= pick <= len(results):
                    break
                print(f"  Enter a number between 1 and {len(results)}")
            except ValueError:
                print("  Enter a valid number")

        # Step 3: Click the result → sends to Saved Messages
        message = await select_and_send_result(client, results, pick - 1)
        if not message:
            print("[ABORT] No audio file received. Exiting.")
            return

        # Step 3: Download the audio file
        file_path = await download_audio(client, message)
        if not file_path:
            print("[ABORT] Download failed. Exiting.")
            return

        # Step 4: Extract metadata
        meta_artist, meta_title = extract_metadata(file_path)

        # Step 5: Fetch lyrics (synced preferred, plain as fallback)
        lyrics, is_synced = search_lrclib(meta_artist, meta_title)
        if not lyrics:
            # Retry with user-provided names if metadata didn't match
            if (meta_artist.lower() != artist_input.lower()
                    or meta_title.lower() != title_input.lower()):
                print("[RETRY] Trying LRCLIB with original input...")
                lyrics, is_synced = search_lrclib(artist_input, title_input)

        # Step 6: Romanize lyrics if CJK
        romaji_lyrics = None
        if lyrics:
            romaji_lyrics = romanize_lyrics(lyrics)

        # Step 7: Embed lyrics + save lyrics file
        if lyrics:
            chosen = choose_lyrics_format(lyrics, romaji_lyrics)
            embed_lyrics(file_path, chosen)
            save_lrc_file(file_path, chosen, is_synced)
        else:
            print("[DONE] No lyrics to embed.")

        print()
        print(f"[DONE] File ready: {file_path}")

    finally:
        await client.disconnect()
        print("[OK] Disconnected from Telegram")


if __name__ == "__main__":
    asyncio.run(main())

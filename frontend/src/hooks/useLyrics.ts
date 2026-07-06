"use client";

import { useState } from "react";

/**
 * Stub hook — provides the lyrics feature API that the SpotiFLAC App component
 * expects. Returns no-ops and empty state. Will be replaced with real
 * implementation when LRCLIB lyrics download is wired in.
 */
export function useLyrics() {
  const [downloadingLyricsTrack, setDownloadingLyricsTrack] = useState<
    string | null
  >(null);
  const [downloadedLyrics] = useState(() => new Set<string>());
  const [failedLyrics] = useState(() => new Set<string>());
  const [skippedLyrics] = useState(() => new Set<string>());
  const [isBulkDownloadingLyrics] = useState(false);

  function resetLyricsState() {
    /* noop — stub */
  }

  async function handleDownloadLyrics(
    _spotifyId: string,
    _name: string,
    _artists: string,
    _albumName?: string,
    _playlistName?: string,
    _position?: number,
    _albumArtist?: string,
    _releaseDate?: string,
    _discNumber?: number,
    _isAlbum?: boolean,
  ) {
    /* noop — stub */
  }

  return {
    resetLyricsState,
    downloadingLyricsTrack,
    downloadedLyrics,
    failedLyrics,
    skippedLyrics,
    isBulkDownloadingLyrics,
    handleDownloadLyrics,
  };
}

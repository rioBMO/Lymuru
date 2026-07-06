"use client";

import { useState } from "react";

/**
 * Stub hook — provides the cover art download API that the SpotiFLAC App
 * component expects. Returns no-ops and empty state. Will be replaced with
 * real implementation when cover art downloading is wired in.
 */
export function useCover() {
  const [downloadingCoverTrack, setDownloadingCoverTrack] = useState<
    string | null
  >(null);
  const [downloadedCovers] = useState(() => new Set<string>());
  const [failedCovers] = useState(() => new Set<string>());
  const [skippedCovers] = useState(() => new Set<string>());
  const [isBulkDownloadingCovers] = useState(false);

  function resetCoverState() {
    /* noop — stub */
  }

  async function handleDownloadCover(
    _coverUrl: string,
    _trackName: string,
    _artistName: string,
    _albumName?: string,
    _playlistName?: string,
    _position?: number,
    _trackId?: string,
    _albumArtist?: string,
    _releaseDate?: string,
    _discNumber?: number,
    _isAlbum?: boolean,
  ) {
    /* noop — stub */
  }

  return {
    resetCoverState,
    downloadingCoverTrack,
    downloadedCovers,
    failedCovers,
    skippedCovers,
    isBulkDownloadingCovers,
    handleDownloadCover,
  };
}

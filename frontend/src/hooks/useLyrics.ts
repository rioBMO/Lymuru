import { useState } from "react";
import { downloadLyrics } from "@/lib/api";
import { getSettings } from "@/lib/settings";
import { buildTrackFileFields, filenameTemplateFor, type OutputOptions, type TrackFields } from "@/lib/output-path";
import { runBulk, type BulkOutcome } from "@/lib/bulk-progress";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import { resolveTemplateISRC } from "@/hooks/useDownload";
import type { TrackMetadata } from "@/types/api";

type Result = { outcome: BulkOutcome; message: string };

// Saves a track's lyrics as an .lrc file next to its audio file, named like the audio.
async function saveLyrics(spotifyId: string, track: TrackFields, durationMs: number | undefined, opts: OutputOptions): Promise<Result> {
    try {
        const settings = getSettings();
        const isrc = track.isrc || await resolveTemplateISRC({ folderTemplate: settings.folderTemplate, filenameTemplate: filenameTemplateFor(settings, opts.mode) }, spotifyId);
        const response = await downloadLyrics({ spotify_id: spotifyId, duration_ms: durationMs, ...buildTrackFileFields(settings, track, { ...opts, isrc }) });
        if (response.not_found) {
            return { outcome: "skipped", message: "No lyrics found for this track" };
        }
        if (!response.success) {
            return { outcome: "failed", message: response.error || "Failed to download lyrics" };
        }
        if (response.already_exists) {
            return { outcome: "skipped", message: "Lyrics file already exists" };
        }
        return { outcome: "done", message: "Lyrics downloaded" };
    }
    catch (err) {
        return { outcome: "failed", message: err instanceof Error ? err.message : String(err) };
    }
}

export function useLyrics() {
    const [downloadingLyricsTrack, setDownloadingLyricsTrack] = useState<string | null>(null);
    const [downloadedLyrics, setDownloadedLyrics] = useState(() => new Set<string>());
    const [failedLyrics, setFailedLyrics] = useState(() => new Set<string>());
    const [skippedLyrics, setSkippedLyrics] = useState(() => new Set<string>());
    const [isBulkDownloadingLyrics, setIsBulkDownloadingLyrics] = useState(false);

    function record(key: string, outcome: BulkOutcome) {
        const add = (prev: Set<string>) => new Set(prev).add(key);
        const remove = (prev: Set<string>) => { const next = new Set(prev); next.delete(key); return next; };
        if (outcome === "failed") {
            setFailedLyrics(add);
            return;
        }
        setFailedLyrics(remove);
        setDownloadedLyrics(add);
        if (outcome === "skipped") {
            setSkippedLyrics(add);
        }
    }

    function resetLyricsState() {
        setDownloadingLyricsTrack(null);
        setDownloadedLyrics(new Set());
        setFailedLyrics(new Set());
        setSkippedLyrics(new Set());
    }

    async function handleDownloadLyrics(spotifyId: string, name: string, artists: string, albumName?: string, playlistName?: string, position?: number, albumArtist?: string, releaseDate?: string, discNumber?: number, _isAlbum?: boolean) {
        const key = spotifyId || `${name}-${artists}`;
        setDownloadingLyricsTrack(key);
        const track: TrackFields = { name, artists, album_name: albumName, album_artist: albumArtist, release_date: releaseDate, disc_number: discNumber };
        const { outcome, message } = await saveLyrics(spotifyId, track, undefined, { mode: "single", folderName: playlistName, position });
        record(key, outcome);
        setDownloadingLyricsTrack(null);
        if (outcome === "done") toast.success(message);
        else if (outcome === "skipped") toast.info(message);
        else toast.error(message);
    }

    async function handleDownloadAllLyrics(tracks: TrackMetadata[], folderName?: string, _isArtistDiscography?: boolean, isAlbum?: boolean) {
        if (tracks.length === 0) {
            toast.error("No tracks to download lyrics for");
            return;
        }
        const items = tracks.map((track, index) => ({ track, position: index + 1 }));
        setIsBulkDownloadingLyrics(true);
        try {
            await runBulk(items, "Downloading lyrics", async ({ track, position }) => {
                const key = track.spotify_id || `${track.name}-${track.artists}`;
                setDownloadingLyricsTrack(key);
                const { outcome } = await saveLyrics(track.spotify_id || "", track, track.duration_ms, { mode: "bulk", folderName, isAlbum, position });
                record(key, outcome);
                return outcome;
            });
        }
        finally {
            setDownloadingLyricsTrack(null);
            setIsBulkDownloadingLyrics(false);
        }
    }

    return {
        resetLyricsState,
        downloadingLyricsTrack,
        downloadedLyrics,
        failedLyrics,
        skippedLyrics,
        isBulkDownloadingLyrics,
        handleDownloadLyrics,
        handleDownloadAllLyrics,
    };
}

import { useState } from "react";
import { downloadCover } from "@/lib/api";
import { getSettings } from "@/lib/settings";
import { buildTrackFileFields, filenameTemplateFor, type OutputOptions, type TrackFields } from "@/lib/output-path";
import { runBulk, type BulkOutcome } from "@/lib/bulk-progress";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import { resolveTemplateISRC } from "@/hooks/useDownload";
import type { TrackMetadata } from "@/types/api";

type Result = { outcome: BulkOutcome; message: string };

// Saves a cover image next to each track's audio file, named like the audio.
async function saveCover(coverUrl: string, track: TrackFields, spotifyId: string | undefined, opts: OutputOptions): Promise<Result> {
    try {
        const settings = getSettings();
        const isrc = track.isrc || await resolveTemplateISRC({ folderTemplate: settings.folderTemplate, filenameTemplate: filenameTemplateFor(settings, opts.mode) }, spotifyId);
        const response = await downloadCover({ cover_url: coverUrl, ...buildTrackFileFields(settings, track, { ...opts, isrc }) });
        if (!response.success) {
            return { outcome: "failed", message: response.error || "Failed to download cover" };
        }
        if (response.already_exists) {
            return { outcome: "skipped", message: "Cover already exists" };
        }
        return { outcome: "done", message: "Cover downloaded" };
    }
    catch (err) {
        return { outcome: "failed", message: err instanceof Error ? err.message : String(err) };
    }
}

export function useCover() {
    const [downloadingCoverTrack, setDownloadingCoverTrack] = useState<string | null>(null);
    const [downloadedCovers, setDownloadedCovers] = useState(() => new Set<string>());
    const [failedCovers, setFailedCovers] = useState(() => new Set<string>());
    const [skippedCovers, setSkippedCovers] = useState(() => new Set<string>());
    const [isBulkDownloadingCovers, setIsBulkDownloadingCovers] = useState(false);

    function record(key: string, outcome: BulkOutcome) {
        const add = (prev: Set<string>) => new Set(prev).add(key);
        const remove = (prev: Set<string>) => { const next = new Set(prev); next.delete(key); return next; };
        if (outcome === "failed") {
            setFailedCovers(add);
            return;
        }
        setFailedCovers(remove);
        setDownloadedCovers(add);
        if (outcome === "skipped") {
            setSkippedCovers(add);
        }
    }

    function resetCoverState() {
        setDownloadingCoverTrack(null);
        setDownloadedCovers(new Set());
        setFailedCovers(new Set());
        setSkippedCovers(new Set());
    }

    async function handleDownloadCover(coverUrl: string, trackName: string, artistName: string, albumName?: string, playlistName?: string, position?: number, trackId?: string, albumArtist?: string, releaseDate?: string, discNumber?: number, _isAlbum?: boolean) {
        if (!coverUrl) {
            toast.error("No cover available for this track");
            return;
        }
        const key = trackId || `${trackName}-${artistName}`;
        setDownloadingCoverTrack(key);
        const track: TrackFields = { name: trackName, artists: artistName, album_name: albumName, album_artist: albumArtist, release_date: releaseDate, disc_number: discNumber };
        const { outcome, message } = await saveCover(coverUrl, track, trackId, { mode: "single", folderName: playlistName, position });
        record(key, outcome);
        setDownloadingCoverTrack(null);
        if (outcome === "done") toast.success(message);
        else if (outcome === "skipped") toast.info(message);
        else toast.error(message);
    }

    async function handleDownloadAllCovers(tracks: TrackMetadata[], folderName?: string, isAlbum?: boolean) {
        const items = tracks.map((track, index) => ({ track, position: index + 1 })).filter(({ track }) => track.images);
        if (items.length === 0) {
            toast.error("No covers available to download");
            return;
        }
        setIsBulkDownloadingCovers(true);
        try {
            await runBulk(items, "Downloading covers", async ({ track, position }) => {
                const key = track.spotify_id || `${track.name}-${track.artists}`;
                setDownloadingCoverTrack(key);
                const { outcome } = await saveCover(track.images, track, track.spotify_id, { mode: "bulk", folderName, isAlbum, position });
                record(key, outcome);
                return outcome;
            });
        }
        finally {
            setDownloadingCoverTrack(null);
            setIsBulkDownloadingCovers(false);
        }
    }

    return {
        resetCoverState,
        downloadingCoverTrack,
        downloadedCovers,
        failedCovers,
        skippedCovers,
        isBulkDownloadingCovers,
        handleDownloadCover,
        handleDownloadAllCovers,
    };
}

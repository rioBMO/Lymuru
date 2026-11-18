import { getAlbumCategoryLabel, getEffectiveAlbumFilenameTemplate, parseTemplate, type Settings, type TemplateData } from "./settings";
import { getFirstArtist, joinPath, sanitizePath } from "./utils";
import type { TrackMetadata } from "@/types/api";

// Where a track's cover (.jpg) or lyrics (.lrc) file goes, and how it is
// named, so that it sits next to the track's audio file.
//
// This mirrors the folder logic in hooks/useDownload.ts:
//   "single" = downloadWithAutoFallback (one track's download button)
//   "bulk"   = downloadWithItemID (Download All / Download Selected)
// Keep the two in sync until they share one implementation.

export type OutputMode = "single" | "bulk";

export type TrackFields = Pick<TrackMetadata, "name" | "artists"> &
    Partial<Pick<TrackMetadata, "album_name" | "album_artist" | "release_date" | "track_number" | "disc_number" | "total_tracks" | "total_discs" | "album_type" | "upc" | "isrc">>;

export interface OutputOptions {
    mode: OutputMode;
    folderName?: string; // album, playlist or artist name passed down from App.tsx
    isAlbum?: boolean;
    position?: number; // 1-based position in the list
    isrc?: string;
}

// Fields shared by CoverDownloadRequest and LyricsDownloadRequest.
export interface TrackFileFields {
    track_name: string;
    artist_name: string;
    artists: string;
    album_name: string;
    album_artist: string;
    release_date: string;
    output_dir: string;
    filename_format: string;
    playlist_name?: string;
    category: string;
    upc: string;
    isrc: string;
    track_number: boolean;
    position: number;
    disc_number: number;
    total_tracks: number;
    total_discs: number;
    use_album_track_number: boolean;
}

const SLASH = "__SLASH_PLACEHOLDER__";
const escapeSlash = (value?: string) => value?.replace(/\//g, SLASH);

export function filenameTemplateFor(settings: Settings, mode: OutputMode): string {
    return mode === "bulk" ? getEffectiveAlbumFilenameTemplate(settings) : settings.filenameTemplate;
}

export function buildTrackFileFields(settings: Settings, track: TrackFields, opts: OutputOptions): TrackFileFields {
    const os = settings.operatingSystem;
    const folderTemplate = settings.folderTemplate || "";
    const applyFolderTemplate = folderTemplate.trim() !== "" && (opts.mode === "bulk" || settings.applyFolderToSingleTrack);
    const position = opts.position || 0;
    const displayArtist = settings.useFirstArtistOnly && track.artists ? getFirstArtist(track.artists) : track.artists;
    const displayAlbumArtist = settings.useFirstArtistOnly && track.album_artist ? getFirstArtist(track.album_artist) : track.album_artist;

    let outputDir = settings.downloadPath;
    const usesAlbumSubfolder = ["{album}", "{album_artist}", "{playlist}"].some((token) => folderTemplate.includes(token));
    const addFolderNameDir = opts.mode === "bulk" ? !opts.isAlbum || !usesAlbumSubfolder : !usesAlbumSubfolder;
    if (settings.createPlaylistFolder && opts.folderName && addFolderNameDir) {
        outputDir = joinPath(os, outputDir, sanitizePath(opts.folderName.replace(/\//g, " "), os));
    }
    if (applyFolderTemplate) {
        const templateData: TemplateData = {
            artist: escapeSlash(displayArtist),
            artists: escapeSlash(track.artists),
            album: escapeSlash(track.album_name),
            album_artist: escapeSlash(displayAlbumArtist) || escapeSlash(displayArtist),
            title: escapeSlash(track.name),
            isrc: escapeSlash(opts.isrc),
            track: track.track_number && track.track_number > 0 ? track.track_number : position,
            total_tracks: track.total_tracks,
            total_discs: track.total_discs,
            year: track.release_date?.substring(0, 4),
            date: track.release_date,
            playlist: escapeSlash(opts.folderName),
        };
        const folderPath = parseTemplate(folderTemplate, templateData);
        for (const part of folderPath.split("/").filter((p) => p.trim())) {
            outputDir = joinPath(os, outputDir, sanitizePath(part.replace(new RegExp(SLASH, "g"), " "), os));
        }
    }

    return {
        track_name: track.name,
        artist_name: displayArtist || "",
        artists: track.artists || "",
        album_name: track.album_name || "",
        album_artist: displayAlbumArtist || "",
        release_date: track.release_date || "",
        output_dir: outputDir,
        filename_format: filenameTemplateFor(settings, opts.mode) || "",
        playlist_name: opts.folderName,
        category: getAlbumCategoryLabel(track.album_type),
        upc: track.upc || "",
        isrc: opts.isrc || "",
        track_number: settings.trackNumber,
        position,
        disc_number: track.disc_number || 0,
        total_tracks: track.total_tracks || 0,
        total_discs: track.total_discs || 0,
        use_album_track_number: opts.mode === "single" && applyFolderTemplate,
    };
}

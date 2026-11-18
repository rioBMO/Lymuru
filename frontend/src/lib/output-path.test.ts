import { describe, expect, test } from "bun:test";

// settings.ts reads window.navigator when it loads; bun test has no window.
(globalThis as any).window ??= { navigator: { platform: "linux" } };
const { buildTrackFileFields } = await import("./output-path");
const { DEFAULT_SETTINGS } = await import("./settings");

const settings = (overrides: Record<string, unknown> = {}) => ({
    ...DEFAULT_SETTINGS,
    downloadPath: "/music",
    operatingSystem: "linux/MacOS" as const,
    folderTemplate: "",
    createPlaylistFolder: false,
    applyFolderToSingleTrack: false,
    ...overrides,
});

const track = {
    name: "Song",
    artists: "Artist",
    album_name: "Album",
    album_artist: "Album Artist",
    release_date: "2020-05-01",
    track_number: 3,
};

describe("buildTrackFileFields output_dir", () => {
    test("uses the downloads folder when no folder options are set", () => {
        expect(buildTrackFileFields(settings(), track, { mode: "bulk" }).output_dir).toBe("/music");
    });

    test("bulk playlist: playlist folder, then the folder template", () => {
        const s = settings({ createPlaylistFolder: true, folderTemplate: "{album_artist}/{album}" });
        const fields = buildTrackFileFields(s, track, { mode: "bulk", folderName: "Road Trip" });
        expect(fields.output_dir).toBe("/music/Road Trip/Album Artist/Album");
    });

    test("bulk album with {album} in the template adds no extra folder", () => {
        const s = settings({ createPlaylistFolder: true, folderTemplate: "{album_artist}/{album}" });
        const fields = buildTrackFileFields(s, track, { mode: "bulk", folderName: "Album", isAlbum: true });
        expect(fields.output_dir).toBe("/music/Album Artist/Album");
    });

    test("a slash inside a name becomes a space, not a subfolder", () => {
        const s = settings({ folderTemplate: "{album_artist}" });
        const fields = buildTrackFileFields(s, { ...track, album_artist: "AC/DC" }, { mode: "bulk" });
        expect(fields.output_dir).toBe("/music/AC DC");
    });

    test("single mode ignores the folder template unless applyFolderToSingleTrack is on", () => {
        const s = settings({ folderTemplate: "{album}" });
        expect(buildTrackFileFields(s, track, { mode: "single" }).output_dir).toBe("/music");
        const on = settings({ folderTemplate: "{album}", applyFolderToSingleTrack: true });
        expect(buildTrackFileFields(on, track, { mode: "single" }).output_dir).toBe("/music/Album");
    });

    test("playlist folder name with a slash is flattened", () => {
        const s = settings({ createPlaylistFolder: true });
        const fields = buildTrackFileFields(s, track, { mode: "bulk", folderName: "Rock/Pop" });
        expect(fields.output_dir).toBe("/music/Rock Pop");
    });
});

describe("buildTrackFileFields filename fields", () => {
    test("single mode uses the normal filename template", () => {
        const s = settings({ useSeparateAlbumFilename: true, albumFilenameTemplate: "{track}. {title}" });
        expect(buildTrackFileFields(s, track, { mode: "single" }).filename_format).toBe(s.filenameTemplate);
    });

    test("bulk mode uses the album filename template when it is enabled", () => {
        const s = settings({ useSeparateAlbumFilename: true, albumFilenameTemplate: "{track}. {title}" });
        expect(buildTrackFileFields(s, track, { mode: "bulk" }).filename_format).toBe("{track}. {title}");
    });

    test("useFirstArtistOnly shortens the artist used in names", () => {
        const s = settings({ useFirstArtistOnly: true });
        const fields = buildTrackFileFields(s, { ...track, artists: "A, B" }, { mode: "bulk" });
        expect(fields.artist_name).toBe("A");
        expect(fields.artists).toBe("A, B");
    });
});

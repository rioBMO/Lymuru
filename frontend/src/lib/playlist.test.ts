import { describe, expect, test } from "bun:test";
import { buildPlaylistFolderName } from "./playlist";

describe("buildPlaylistFolderName", () => {
    test("uses the playlist name", () => {
        expect(buildPlaylistFolderName("Road Trip", "alice", false)).toBe("Road Trip");
    });

    test("adds the owner when the setting is on", () => {
        expect(buildPlaylistFolderName("Road Trip", "alice", true)).toBe("Road Trip - alice");
    });

    test("ignores the setting when there is no owner", () => {
        expect(buildPlaylistFolderName("Road Trip", "  ", true)).toBe("Road Trip");
        expect(buildPlaylistFolderName("Road Trip", undefined, true)).toBe("Road Trip");
    });

    test("falls back to a generic name when the name is empty", () => {
        expect(buildPlaylistFolderName("", "alice", false)).toBe("Playlist");
    });
});

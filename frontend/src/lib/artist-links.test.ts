import { describe, expect, test } from "bun:test";
import { buildClickableArtists, getClickableArtistKey, splitArtistNames } from "./artist-links";

describe("splitArtistNames", () => {
    test("splits on comma and semicolon, trims, drops empty", () => {
        expect(splitArtistNames("A, B; C ;")).toEqual(["A", "B", "C"]);
    });

    test("keeps a name with a comma but no space", () => {
        expect(splitArtistNames("Tyler,The Creator")).toEqual(["Tyler,The Creator"]);
    });
});

describe("buildClickableArtists", () => {
    test("uses artists_data and fills a missing URL from the id", () => {
        const result = buildClickableArtists("A, B", [
            { id: "1", name: "A", external_urls: "https://example.com/a" },
            { id: "2", name: "B", external_urls: "" },
        ]);
        expect(result).toEqual([
            { id: "1", name: "A", external_urls: "https://example.com/a" },
            { id: "2", name: "B", external_urls: "https://open.spotify.com/artist/2" },
        ]);
    });

    test("single artist with an id gets a Spotify URL", () => {
        expect(buildClickableArtists("A", undefined, "1")).toEqual([
            { id: "1", name: "A", external_urls: "https://open.spotify.com/artist/1" },
        ]);
    });

    test("single artist prefers the given URL", () => {
        expect(buildClickableArtists("A", [], "1", "https://example.com/a")).toEqual([
            { id: "1", name: "A", external_urls: "https://example.com/a" },
        ]);
    });

    test("several names without data are separate, unresolved links", () => {
        expect(buildClickableArtists("A, B", undefined, "1")).toEqual([
            { id: "", name: "A", external_urls: "" },
            { id: "", name: "B", external_urls: "" },
        ]);
    });
});

describe("getClickableArtistKey", () => {
    test("uses the id, falling back to the name", () => {
        expect(getClickableArtistKey({ id: "1", name: "A", external_urls: "" })).toBe("1");
        expect(getClickableArtistKey({ id: "", name: "A", external_urls: "" })).toBe("A");
    });
});

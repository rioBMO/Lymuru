import type { ArtistSimple } from "@/types/api";

const spotifyArtistUrl = (id: string) => `https://open.spotify.com/artist/${id}`;

// Splits an artist string such as "A, B" or "A; B" into trimmed names.
export function splitArtistNames(artists: string): string[] {
    return (artists ?? "")
        .split(/;|, /)
        .map((name) => name.trim())
        .filter((name) => name !== "");
}

// Returns one entry per artist. An empty external_urls means "resolve by
// searching for the name" (see useMetadata.handleArtistClick).
export function buildClickableArtists(artists: string, data?: ArtistSimple[], artistId?: string, artistUrl?: string): ArtistSimple[] {
    if (data && data.length > 0) {
        return data.map((artist) => artist.id && !artist.external_urls
            ? { ...artist, external_urls: spotifyArtistUrl(artist.id) }
            : artist);
    }
    const names = splitArtistNames(artists);
    if (names.length === 1 && (artistId || artistUrl)) {
        return [{
            id: artistId ?? "",
            name: names[0],
            external_urls: artistUrl || spotifyArtistUrl(artistId!),
        }];
    }
    return names.map((name) => ({ id: "", name, external_urls: "" }));
}

export function getClickableArtistKey(artist: ArtistSimple): string {
    return artist.id || artist.name;
}

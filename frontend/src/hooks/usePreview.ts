export function usePreview() {
    return {
        playPreview: (_trackId: string, _trackName: string) => {},
        loadingPreview: null as string | null,
        playingTrack: null as string | null,
    };
}

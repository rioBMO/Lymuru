export function usePreview() {
    return {
        previewId: null,
        isPlaying: false,
        togglePreview: (id: string, url: string) => {},
        pausePreview: () => {},
        volume: 0.5,
        setVolume: (v: number) => {}
    };
}

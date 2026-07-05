// Stub: preview player for History page inline playback.
// To be replaced with a real audio playback implementation later.
export interface PreviewPlayback {
    audio: HTMLAudioElement;
    destroy(): void;
}

export async function createPreviewPlayback(url: string, volume?: number): Promise<PreviewPlayback> {
    const audio = new Audio(url);
    if (volume !== undefined) {
        audio.volume = Math.max(0, Math.min(1, volume));
    }
    return {
        audio,
        destroy() {
            audio.pause();
            audio.src = "";
        },
    };
}

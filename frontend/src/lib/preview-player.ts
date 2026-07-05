// Preview player with single-instance management, play/pause/seek/volume.
// Only one preview can play at a time — starting a new one stops the previous.

import { getPreviewVolume } from "@/lib/preview";

export interface PreviewPlayback {
    readonly url: string;
    play(): void;
    pause(): void;
    seek(time: number): void;
    setVolume(volume: number): void;
    getCurrentTime(): number;
    getDuration(): number;
    isPlaying(): boolean;
    onTimeUpdate(cb: (time: number) => void): () => void;
    onEnded(cb: () => void): () => void;
    onLoaded(cb: (duration: number) => void): () => void;
    onError(cb: (err: string) => void): () => void;
    destroy(): void;
}

let currentInstance: PreviewPlayback | null = null;

export async function createPreviewPlayback(url: string, initialVolume?: number): Promise<PreviewPlayback> {
    // Stop the previous instance.
    if (currentInstance) {
        currentInstance.destroy();
        currentInstance = null;
    }

    const audio = new Audio(url);
    audio.preload = "metadata";
    audio.volume = initialVolume !== undefined
        ? Math.max(0, Math.min(1, initialVolume))
        : Math.max(0, Math.min(1, getPreviewVolume()));

    const listeners: Map<string, Array<(...args: any[]) => void>> = new Map();

    const emit = (event: string, ...args: any[]) => {
        const cbs = listeners.get(event);
        if (cbs) cbs.forEach((cb) => cb(...args));
    };

    const on = (event: string, cb: (...args: any[]) => void): (() => void) => {
        if (!listeners.has(event)) listeners.set(event, []);
        listeners.get(event)!.push(cb);
        return () => {
            const arr = listeners.get(event);
            if (arr) {
                const idx = arr.indexOf(cb);
                if (idx >= 0) arr.splice(idx, 1);
            }
        };
    };

    audio.addEventListener("timeupdate", () => emit("timeupdate", audio.currentTime));
    audio.addEventListener("ended", () => emit("ended"));
    audio.addEventListener("loadedmetadata", () => emit("loaded", audio.duration));
    audio.addEventListener("error", () => {
        let msg = "Failed to load preview";
        if (audio.error) {
            switch (audio.error.code) {
                case MediaError.MEDIA_ERR_NETWORK: msg = "Network error loading preview"; break;
                case MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED: msg = "Preview format not supported"; break;
            }
        }
        emit("error", msg);
    });

    const instance: PreviewPlayback = {
        url,
        play() { audio.play().catch(() => {}); },
        pause() { audio.pause(); },
        seek(time: number) { audio.currentTime = Math.max(0, Math.min(time, audio.duration || 0)); },
        setVolume(volume: number) { audio.volume = Math.max(0, Math.min(1, volume)); },
        getCurrentTime() { return audio.currentTime; },
        getDuration() { return audio.duration || 0; },
        isPlaying() { return !audio.paused; },
        onTimeUpdate(cb) { return on("timeupdate", cb); },
        onEnded(cb) { return on("ended", cb); },
        onLoaded(cb) { return on("loaded", cb); },
        onError(cb) { return on("error", cb); },
        destroy() {
            audio.pause();
            audio.src = "";
            listeners.clear();
            if (currentInstance === instance) currentInstance = null;
        },
    };

    currentInstance = instance;
    return instance;
}

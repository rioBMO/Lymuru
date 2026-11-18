import { useEffect, useRef, useState } from "react";
import { GetPreviewURL } from "../../wailsjs/go/main/App";
import { createPreviewPlayback, type PreviewPlayback } from "@/lib/preview-player";
import { getPreviewVolume } from "@/lib/preview";
import { toastWithSound as toast } from "@/lib/toast-with-sound";

// Plays a track's Spotify preview. Clicking the playing track again stops it.
export function usePreview() {
    const [loadingPreview, setLoadingPreview] = useState<string | null>(null);
    const [playingTrack, setPlayingTrack] = useState<string | null>(null);
    const playbackRef = useRef<PreviewPlayback | null>(null);

    const stop = () => {
        playbackRef.current?.destroy();
        playbackRef.current = null;
        setPlayingTrack(null);
    };

    // Stop playback when the page that owns it goes away.
    useEffect(() => () => playbackRef.current?.destroy(), []);

    const playPreview = async (trackId: string, trackName: string) => {
        if (playingTrack === trackId) {
            stop();
            return;
        }
        stop();
        setLoadingPreview(trackId);
        try {
            const url = await GetPreviewURL(trackId);
            if (!url) {
                toast.error(`No preview available for ${trackName}`);
                return;
            }
            const playback = await createPreviewPlayback(url, getPreviewVolume());
            playbackRef.current = playback;
            const finish = () => {
                if (playbackRef.current === playback) {
                    playback.destroy();
                    playbackRef.current = null;
                    setPlayingTrack(null);
                }
            };
            playback.onEnded(finish);
            playback.onError((message) => {
                finish();
                toast.error(message);
            });
            playback.play();
            setPlayingTrack(trackId);
        }
        catch (error) {
            toast.error(`Failed to play preview: ${error}`);
        }
        finally {
            setLoadingPreview(null);
        }
    };

    return { playPreview, loadingPreview, playingTrack };
}

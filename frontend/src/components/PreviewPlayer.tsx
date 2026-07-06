import { useEffect, useRef, useState, useCallback } from "react";
import { Play, Pause, Volume2 } from "lucide-react";
import { createPreviewPlayback, type PreviewPlayback } from "@/lib/preview-player";

interface PreviewPlayerProps {
    url: string;
}

export function PreviewPlayer({ url }: PreviewPlayerProps) {
    const [playing, setPlaying] = useState(false);
    const [currentTime, setCurrentTime] = useState(0);
    const [duration, setDuration] = useState(0);
    const [volume, setVolume] = useState(1);
    const [error, setError] = useState("");
    const [loaded, setLoaded] = useState(false);
    const playbackRef = useRef<PreviewPlayback | null>(null);
    const cleanupsRef = useRef<Array<() => void>>([]);

    const cleanup = useCallback(() => {
        cleanupsRef.current.forEach((fn) => fn());
        cleanupsRef.current = [];
        if (playbackRef.current) {
            playbackRef.current.destroy();
            playbackRef.current = null;
        }
    }, []);

    useEffect(() => {
        cleanup();
        setPlaying(false);
        setCurrentTime(0);
        setDuration(0);
        setError("");
        setLoaded(false);

        createPreviewPlayback(url).then((pb) => {
            playbackRef.current = pb;
            cleanupsRef.current.push(
                pb.onTimeUpdate((t) => setCurrentTime(t)),
                pb.onEnded(() => setPlaying(false)),
                pb.onLoaded((d) => { setDuration(d); setLoaded(true); }),
                pb.onError((err) => setError(err)),
            );
            pb.play();
            setPlaying(true);
        }).catch((err) => {
            setError(err instanceof Error ? err.message : "Failed to create player");
        });

        return cleanup;
    }, [url, cleanup]);

    const togglePlay = () => {
        if (!playbackRef.current) return;
        if (playing) {
            playbackRef.current.pause();
            setPlaying(false);
        } else {
            playbackRef.current.play();
            setPlaying(true);
        }
    };

    const handleSeek = (pct: number) => {
        if (playbackRef.current && duration > 0) {
            playbackRef.current.seek((pct / 100) * duration);
        }
    };

    const handleVolumeChange = (v: number) => {
        setVolume(v);
        playbackRef.current?.setVolume(v);
    };

    const formatTime = (s: number) => {
        const m = Math.floor(s / 60);
        const sec = Math.floor(s % 60);
        return `${m}:${sec.toString().padStart(2, "0")}`;
    };

    const progressPct = duration > 0 ? (currentTime / duration) * 100 : 0;

    return (
        <div className="flex items-center gap-2 rounded-md border p-2 text-xs">
            <button
                onClick={togglePlay}
                className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-primary-foreground hover:bg-primary/80"
                disabled={!!error}
            >
                {playing ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
            </button>

            {error ? (
                <span className="text-destructive truncate">{error}</span>
            ) : loaded ? (
                <>
                    <span className="text-muted-foreground w-10 text-right tabular-nums">{formatTime(currentTime)}</span>
                    <div
                        className="flex-1 h-1 bg-muted rounded-full cursor-pointer relative"
                        onClick={(e) => {
                            const rect = e.currentTarget.getBoundingClientRect();
                            const pct = ((e.clientX - rect.left) / rect.width) * 100;
                            handleSeek(pct);
                        }}
                    >
                        <div className="h-full bg-primary rounded-full" style={{ width: `${progressPct}%` }} />
                    </div>
                    <span className="text-muted-foreground w-10 tabular-nums">{formatTime(duration)}</span>
                </>
            ) : (
                <span className="text-muted-foreground">Loading preview...</span>
            )}

            <div className="flex items-center gap-1">
                <Volume2 className="h-3 w-3 text-muted-foreground" />
                <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.05"
                    value={volume}
                    onChange={(e) => handleVolumeChange(parseFloat(e.target.value))}
                    className="w-12 h-1"
                />
            </div>
        </div>
    );
}

import { useState, useEffect, useRef } from "react";
import { Clock, X } from "lucide-react";
import { useDownloadProgress } from "@/hooks/useDownloadProgress";

const DISMISSED_KEY = "lymuru_cooldown_dismissed_event";

function formatCountdown(totalSecs: number): string {
    if (totalSecs <= 0) return "";
    const m = Math.floor(totalSecs / 60);
    const s = totalSecs % 60;
    if (m > 0) return `${m}m ${s}s`;
    return `${s}s`;
}

export function CooldownBanner() {
    const progress = useDownloadProgress();
    const isCooldown = Boolean(progress.cooldown) && (progress.cooldown_secs ?? 0) > 0;
    const eventID = progress.cooldown_event_id ?? 0;
    const provider = progress.cooldown_provider?.trim();
    const isDownloading = progress.is_downloading;

    const [dismissedEventID, setDismissedEventID] = useState(() => {
        const stored = Number(localStorage.getItem(DISMISSED_KEY));
        return Number.isFinite(stored) ? stored : 0;
    });
    const [countdown, setCountdown] = useState(0);
    const timerRef = useRef<number | null>(null);

    useEffect(() => {
        if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
        }
        if (!isCooldown || !progress.cooldown_until) {
            setCountdown(0);
            return;
        }
        const tick = () => {
            const remaining = Math.max(0, Math.ceil((progress.cooldown_until! - Date.now()) / 1000));
            setCountdown(remaining);
        };
        tick();
        timerRef.current = window.setInterval(tick, 1000);
        return () => {
            if (timerRef.current) {
                clearInterval(timerRef.current);
                timerRef.current = null;
            }
        };
    }, [isCooldown, progress.cooldown_until]);

    const dismiss = (id: number) => {
        setDismissedEventID(id);
        localStorage.setItem(DISMISSED_KEY, String(id));
    };

    // Suppress banner while a download is active (auto-fallback in progress).
    if (!isCooldown || dismissedEventID === eventID || isDownloading) {
        return null;
    }

    const message = progress.cooldown_message?.trim();
    const countdownStr = formatCountdown(countdown);
    const displayMsg = provider
        ? `${provider}: ${message || "API on cooldown"}`
        : (message || "API on cooldown");

    return (
        <div className="fixed top-12 left-1/2 z-50 -translate-x-1/2 animate-in fade-in slide-in-from-top-2">
            <div className="flex items-center gap-2.5 rounded-lg border border-amber-300 bg-amber-50 px-3.5 py-2 text-amber-700 shadow-lg dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300">
                <Clock className="h-4 w-4 shrink-0" />
                <p className="text-xs font-medium leading-tight">
                    {displayMsg}
                    {countdownStr && (
                        <span className="ml-1 tabular-nums opacity-80">
                            (retry in {countdownStr})
                        </span>
                    )}
                </p>
                <button
                    type="button"
                    onClick={() => dismiss(eventID)}
                    aria-label="Dismiss"
                    className="ml-1 shrink-0 rounded-md p-1 text-amber-600/70 transition-colors hover:bg-amber-100 hover:text-amber-700 dark:text-amber-400/70 dark:hover:bg-amber-900 dark:hover:text-amber-300"
                >
                    <X className="h-3.5 w-3.5" />
                </button>
            </div>
        </div>
    );
}

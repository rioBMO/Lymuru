import { toast } from "sonner";
import { toastWithSound } from "./toast-with-sound";

export type BulkOutcome = "done" | "skipped" | "failed";

export interface BulkSummary {
    done: number;
    skipped: number;
    failed: number;
    stopped: boolean;
}

let nextId = 0;

// Runs `work` for each item in order, with one progress toast that has a Stop
// button. Stop takes effect after the current item. Ends with a summary toast.
export async function runBulk<T>(items: T[], label: string, work: (item: T, index: number) => Promise<BulkOutcome>): Promise<BulkSummary> {
    const id = `bulk-${++nextId}`;
    const summary: BulkSummary = { done: 0, skipped: 0, failed: 0, stopped: false };
    const stop = () => { summary.stopped = true; };

    for (let i = 0; i < items.length && !summary.stopped; i++) {
        // Plain sonner toast: no sound on every progress update.
        toast.loading(`${label} ${i + 1}/${items.length}`, { id, duration: Infinity, action: { label: "Stop", onClick: stop } });
        try {
            summary[await work(items[i], i)]++;
        }
        catch {
            summary.failed++;
        }
    }
    toast.dismiss(id);

    const counts = `${summary.done} downloaded, ${summary.skipped} skipped, ${summary.failed} failed`;
    const message = `${label}${summary.stopped ? " stopped" : ""}: ${counts}`;
    if (summary.failed > 0) {
        toastWithSound.warning(message);
    }
    else {
        toastWithSound.success(message);
    }
    return summary;
}

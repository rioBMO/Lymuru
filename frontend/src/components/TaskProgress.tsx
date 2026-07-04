import { formatSize } from "@/lib/format";
import { useQueue } from "@/context/QueueContext";
import { Card, CardContent } from "@/components/ui/card";

interface Props {
  taskId?: string;
}

/**
 * TaskProgress shows the current state of a single in-flight task.
 * It reads from the global queue (which is fed by Wails events)
 * using the provided taskId, or shows the first active task.
 */
export function TaskProgress({ taskId }: Props) {
  const queue = useQueue();
  const task = taskId
    ? queue.tasks.find((t) => t.task_id === taskId)
    : queue.tasks.find((t) => t.phase !== "complete" && !t.error);

  if (!task) {
    return (
      <Card>
        <CardContent className="p-5 text-center text-sm text-muted-foreground">
          <img
            src="/assets/image/lymuru-mascot.png"
            alt="Working…"
            className="h-12 w-auto mx-auto animate-bounce"
          />
          <p className="mt-2">Working…</p>
        </CardContent>
      </Card>
    );
  }

  const pct = Math.round(task.download_percent || 0);
  const isDownloading = task.phase === "downloading";

  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex justify-center mb-3">
          <img
            src="/assets/image/lymuru-mascot.png"
            alt="Working…"
            className={`h-16 w-auto ${isDownloading ? "animate-bounce" : ""}`}
          />
        </div>

        <p className="text-center text-sm font-semibold text-foreground mb-2">
          {task.stage || "Working…"}
        </p>

        {isDownloading && (
          <div className="h-2 rounded-full bg-muted overflow-hidden">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${pct}%` }}
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

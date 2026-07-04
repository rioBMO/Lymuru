import { useState, useCallback, useEffect } from "react";
import { useQueue } from "@/context/QueueContext";

// Local FileInfo shape (matches what DownloadFiles expects).
export interface FileInfo {
  filename: string;
  url: string;
  size: number;
}

export type TaskState =
  | { status: "idle" }
  | { status: "running"; taskId: string }
  | { status: "choosing"; taskId: string }
  | { status: "done"; files: FileInfo[]; taskId: string }
  | { status: "error"; message: string; taskId: string };

/**
 * Local state hook for a single tab's task.
 *
 * In the Wails architecture, the Go side maintains the authoritative
 * task state and emits `task:progress`, `task:complete`, and
 * `task:error` events. The QueueContext subscribes to those events
 * and exposes the latest known task list.
 *
 * This hook just observes the queue for a specific task ID and
 * translates those events into a tab-local state shape.
 */
export function useTaskProgress() {
  const queue = useQueue();
  const [state, setState] = useState<TaskState>({ status: "idle" });
  const [currentTaskId, setCurrentTaskId] = useState<string | null>(null);

  useEffect(() => {
    if (!currentTaskId) return;
    const task = queue.tasks.find((t) => t.task_id === currentTaskId);
    if (!task) return;

    if (task.error) {
      setState({ status: "error", message: task.error, taskId: currentTaskId });
    } else if (task.files && task.files.length > 0) {
      setState({
        status: "done",
        files: task.files.map((p) => ({ filename: p.split(/[\\/]/).pop() ?? p, url: p, size: 0 })),
        taskId: currentTaskId,
      });
    } else if (task.phase === "choosing") {
      setState({ status: "choosing", taskId: currentTaskId });
    } else if (task.phase !== "complete") {
      setState({ status: "running", taskId: currentTaskId });
    }
  }, [queue.tasks, currentTaskId]);

  const start = useCallback(
    (taskId: string) => {
      setCurrentTaskId(taskId);
      queue.attachTask(taskId);
      setState({ status: "running", taskId });
    },
    [queue],
  );

  const reset = useCallback(() => {
    setCurrentTaskId(null);
    setState({ status: "idle" });
  }, []);

  return { state, start, reset };
}

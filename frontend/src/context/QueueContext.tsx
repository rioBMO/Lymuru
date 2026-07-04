import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
import {
  GetActiveTasks,
  CancelTask as apiCancelTask,
  Events,
  type ActiveTask,
  type TaskProgressPayload,
  type TaskCompletePayload,
  type TaskErrorPayload,
} from "@/lib/api";

export interface TaskWithProgress extends ActiveTask {
  progress?: TaskProgressPayload;
  files?: string[];
}

interface QueueState {
  tasks: TaskWithProgress[];
  activeCount: number;
  isQueueOpen: boolean;
  openQueue: () => void;
  closeQueue: () => void;
  cancel: (taskId: string) => Promise<void>;
  clearAll: () => Promise<void>;
  attachTask: (taskId: string) => void;
}

const QueueContext = createContext<QueueState | null>(null);

export function QueueProvider({ children }: { children: ReactNode }) {
  const [tasks, setTasks] = useState<TaskWithProgress[]>([]);
  const [isQueueOpen, setQueueOpen] = useState(false);

  const upsertTask = useCallback((patch: Partial<TaskWithProgress> & { task_id: string }) => {
    setTasks((prev) => {
      const idx = prev.findIndex((t) => t.task_id === patch.task_id);
      if (idx === -1) {
        return [
          ...prev,
          {
            task_id: patch.task_id,
            stage: patch.stage ?? "Starting…",
            phase: patch.phase ?? "preparing",
            task_type: patch.task_type ?? "unknown",
            query: patch.query ?? "",
            download_percent: patch.download_percent ?? 0,
            created_at: patch.created_at ?? new Date().toISOString(),
          },
        ];
      }
      const next = prev.slice();
      next[idx] = { ...next[idx], ...patch };
      return next;
    });
  }, []);

  // Subscribe to Wails events.
  useEffect(() => {
    const onProgress = (...args: unknown[]) => {
      const data = args[0] as TaskProgressPayload | undefined;
      if (!data) return;
      upsertTask({
        task_id: data.task_id,
        stage: data.stage,
        phase: data.phase,
        download_percent: data.download_percent,
        task_type: data.task_type,
        query: data.query,
        progress: data,
      });
    };
    const onComplete = (...args: unknown[]) => {
      const data = args[0] as TaskCompletePayload | undefined;
      if (!data) return;
      upsertTask({
        task_id: data.task_id,
        phase: "complete",
        stage: "Complete",
        download_percent: 100,
        files: data.files,
      });
    };
    const onError = (...args: unknown[]) => {
      const data = args[0] as TaskErrorPayload | undefined;
      if (!data) return;
      upsertTask({
        task_id: data.task_id,
        error: data.message,
      });
    };
    EventsOn(Events.TaskProgress, onProgress as (...args: unknown[]) => void);
    EventsOn(Events.TaskComplete, onComplete as (...args: unknown[]) => void);
    EventsOn(Events.TaskError, onError as (...args: unknown[]) => void);
    return () => {
      EventsOff(Events.TaskProgress);
      EventsOff(Events.TaskComplete);
      EventsOff(Events.TaskError);
    };
  }, [upsertTask]);

  // Initial discovery of any pre-existing active tasks.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await GetActiveTasks();
        if (cancelled) return;
        const existing = (res as unknown) as TaskWithProgress[];
        for (const t of existing) {
          upsertTask(t);
        }
      } catch {
        /* ignore — sidecar may not be ready yet */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [upsertTask]);

  const cancel = useCallback(async (taskId: string) => {
    try {
      await apiCancelTask(taskId);
    } catch {
      /* ignore */
    }
    setTasks((prev) => prev.filter((t) => t.task_id !== taskId));
  }, []);

  const clearAll = useCallback(async () => {
    // Cancel every active task in turn; the queue state is then cleared.
    const ids = tasks
      .filter((t) => t.phase !== "complete" && !t.error)
      .map((t) => t.task_id);
    await Promise.all(ids.map((id) => apiCancelTask(id).catch(() => {})));
    setTasks([]);
  }, [tasks]);

  const openQueue = useCallback(() => setQueueOpen(true), []);
  const closeQueue = useCallback(() => setQueueOpen(false), []);

  // The Go side already maintains task state and emits events. From the
  // frontend's perspective, attachTask is a no-op — the Go side knows
  // about the task and will emit progress events.
  const attachTask = useCallback((_taskId: string) => {
    /* no-op */
  }, []);

  const activeCount = useMemo(
    () => tasks.filter((t) => t.phase !== "complete" && !t.error).length,
    [tasks],
  );

  const value: QueueState = {
    tasks,
    activeCount,
    isQueueOpen,
    openQueue,
    closeQueue,
    cancel,
    clearAll,
    attachTask,
  };

  return <QueueContext.Provider value={value}>{children}</QueueContext.Provider>;
}

export function useQueue(): QueueState {
  const ctx = useContext(QueueContext);
  if (!ctx) throw new Error("useQueue must be used within QueueProvider");
  return ctx;
}

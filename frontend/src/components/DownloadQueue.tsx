import { Clock, X } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Spinner } from "@/components/ui/spinner";
import type { ActiveTask } from "@/lib/api";

interface Props {
  open: boolean;
  tasks: ActiveTask[];
  onClose: () => void;
  onCancel: (taskId: string) => void;
  onClearAll?: () => void;
}

const PHASE_LABELS: Record<string, string> = {
  preparing: "Preparing",
  downloading: "Downloading",
  finalizing: "Finalizing",
  choosing: "Choosing lyrics",
  complete: "Complete",
};

function TaskRow({
  task,
  onCancel,
}: {
  task: ActiveTask;
  onCancel: (id: string) => void;
}) {
  const phase = task.phase;
  const pct = Math.max(0, Math.min(100, task.download_percent || 0));
  const isPreparing = phase === "preparing" || phase === "finalizing";

  return (
    <div className="rounded-lg border border-border bg-muted p-3 space-y-2">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium text-foreground truncate">
            {task.query || task.task_type}
          </p>
          <p className="text-[10px] text-muted-foreground">
            {PHASE_LABELS[phase] ?? phase} — {task.stage}
          </p>
        </div>
        <button
          onClick={() => onCancel(task.task_id)}
          className="shrink-0 inline-flex items-center justify-center rounded-md p-1 text-muted-foreground hover:text-destructive transition-colors"
          title="Cancel task"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="h-1.5 rounded-full bg-background overflow-hidden">
        {phase === "downloading" ? (
          <div
            className="h-full rounded-full bg-primary transition-all duration-300"
            style={{ width: `${pct}%` }}
          />
        ) : isPreparing ? (
          <div className="h-full rounded-full bg-primary/60 animate-pulse" style={{ width: "40%" }} />
        ) : (
          <div className="h-full rounded-full bg-primary" style={{ width: "100%" }} />
        )}
      </div>
    </div>
  );
}

export function DownloadQueue({ open, tasks, onClose, onCancel, onClearAll }: Props) {
  const completed = tasks.filter((t) => t.phase === "complete");
  const active = tasks.filter((t) => t.phase !== "complete");

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Download Queue</DialogTitle>
          <DialogDescription>
            {tasks.length} task{tasks.length === 1 ? "" : "s"} in queue
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="active" className="w-full">
          <div className="flex items-center justify-between mb-2">
            <TabsList>
              <TabsTrigger value="active">
                Active
                {active.length > 0 && (
                  <Badge variant="secondary" className="ml-1">
                    {active.length}
                  </Badge>
                )}
              </TabsTrigger>
              <TabsTrigger value="completed">
                Completed
                {completed.length > 0 && (
                  <Badge variant="success" className="ml-1">
                    {completed.length}
                  </Badge>
                )}
              </TabsTrigger>
            </TabsList>
            {onClearAll && tasks.length > 0 && (
              <Button variant="ghost" size="sm" onClick={onClearAll} className="text-destructive hover:text-destructive">
                Cancel all
              </Button>
            )}
          </div>

          <TabsContent value="active" className="max-h-80 overflow-y-auto space-y-2">
            {active.length === 0 ? (
              <EmptyState message="No active tasks" />
            ) : (
              active.map((t) => <TaskRow key={t.task_id} task={t} onCancel={onCancel} />)
            )}
          </TabsContent>

          <TabsContent value="completed" className="max-h-80 overflow-y-auto space-y-2">
            {completed.length === 0 ? (
              <EmptyState message="No completed tasks" />
            ) : (
              completed.map((t) => <TaskRow key={t.task_id} task={t} onCancel={onCancel} />)
            )}
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center gap-2 py-8 text-muted-foreground text-sm">
      <Clock className="h-8 w-8 opacity-30" />
      <span>{message}</span>
    </div>
  );
}

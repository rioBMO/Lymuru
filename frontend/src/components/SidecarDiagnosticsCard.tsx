import { useEffect, useState } from "react";
import { RefreshCw, Play, Eye, CircleDot, CircleDashed, AlertCircle, ShieldAlert, Circle } from "lucide-react";
import {
  GetSidecarInfo,
  TestSidecar,
  RestartSidecar,
  type SidecarInfo,
} from "@/lib/api";
import { useToast } from "@/components/Toast";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

const STATUS_COLOR: Record<string, string> = {
  online:        "text-emerald-500",
  starting:      "text-muted-foreground",
  stopped:       "text-muted-foreground",
  auth_required: "text-amber-500",
  error:         "text-destructive",
};

const STATUS_ICON: Record<string, typeof CircleDot> = {
  online:        CircleDot,
  starting:      CircleDashed,
  stopped:       Circle,
  auth_required: ShieldAlert,
  error:         AlertCircle,
};

export function SidecarDiagnosticsCard() {
  const { toast } = useToast();
  const [info, setInfo] = useState<SidecarInfo | null>(null);
  const [testing, setTesting] = useState(false);
  const [restarting, setRestarting] = useState(false);
  const [logsOpen, setLogsOpen] = useState(false);

  const refresh = async () => {
    try {
      const i = (await GetSidecarInfo()) as SidecarInfo;
      setInfo(i);
    } catch {
      /* keep last value */
    }
  };

  // Auto-poll every 2s while the card is mounted. This makes the
  // status reflect the real sidecar state even if EventsOn
  // subscription missed a state change.
  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, 2000);
    return () => clearInterval(interval);
  }, []);

  const handleTest = async () => {
    setTesting(true);
    try {
      await TestSidecar();
      toast("Sidecar restart initiated", "success");
      // The poll will pick up the new status automatically.
      setTimeout(refresh, 500);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Test failed", "error");
    }
    setTesting(false);
  };

  const handleRestart = async () => {
    setRestarting(true);
    try {
      await RestartSidecar();
      toast("Sidecar restart initiated", "success");
      setTimeout(refresh, 500);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Restart failed", "error");
    }
    setRestarting(false);
  };

  const status = info?.status ?? "stopped";
  const StatusIcon = STATUS_ICON[status] ?? Circle;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Sidecar Diagnostics</CardTitle>
        <CardDescription>
          The Python sidecar handles Telegram authentication and downloads.
          If it's not online, music features won't work.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {/* Status row */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">Status:</span>
          <span className={cn("flex items-center gap-1 text-sm", STATUS_COLOR[status] ?? "text-muted-foreground")}>
            <StatusIcon className="h-3.5 w-3.5" />
            {status}
          </span>
          {info?.message && (
            <span className="truncate text-xs text-muted-foreground" title={info.message}>
              — {info.message}
            </span>
          )}
        </div>

        {/* Paths */}
        <div className="text-xs text-muted-foreground space-y-0.5">
          <div>Python: <code className="bg-muted px-1 rounded">{info?.python_path || "auto-detect"}</code></div>
          <div>Script: <code className="bg-muted px-1 rounded break-all">{info?.script_dir || "unknown"}</code></div>
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 flex-wrap">
          <Button variant="outline" size="sm" onClick={handleTest} disabled={testing}>
            <Play className="mr-1 h-3.5 w-3.5" />
            {testing ? "Testing…" : "Test Sidecar"}
          </Button>
          <Button variant="outline" size="sm" onClick={handleRestart} disabled={restarting}>
            <RefreshCw className="mr-1 h-3.5 w-3.5" />
            {restarting ? "Restarting…" : "Restart"}
          </Button>
          {(info?.logs?.length ?? 0) > 0 && (
            <Button variant="ghost" size="sm" onClick={() => setLogsOpen(true)}>
              <Eye className="mr-1 h-3.5 w-3.5" />
              View Logs ({info!.logs.length})
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={refresh}>
            <RefreshCw className="mr-1 h-3.5 w-3.5" />
            Refresh
          </Button>
        </div>
      </CardContent>

      {/* Log viewer dialog */}
      <Dialog open={logsOpen} onOpenChange={setLogsOpen}>
        <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>Sidecar Logs</DialogTitle>
            <DialogDescription>
              Recent stderr output from the Python sidecar. Last {info?.logs?.length ?? 0} lines.
            </DialogDescription>
          </DialogHeader>
          <pre className="flex-1 overflow-auto rounded bg-muted p-3 text-xs font-mono whitespace-pre-wrap break-all">
            {info?.logs?.length ? info.logs.map((line, i) => (
              <div key={i} className="leading-relaxed">
                {line}
              </div>
            )) : "No log lines available."}
          </pre>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

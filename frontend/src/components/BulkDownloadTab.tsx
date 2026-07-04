import { useState, useRef, useEffect, useMemo } from "react";
import { DownloadLink, Search as apiSearch, Download as apiDownload } from "@/lib/api";
import type { SearchResponse } from "@/lib/api";
import { useQueue } from "@/context/QueueContext";
import { DownloadFiles, type FileInfo } from "@/components/DownloadFiles";
import { useToast } from "@/components/Toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

function isValidUrl(text: string): boolean {
  try {
    new URL(text);
    return true;
  } catch {
    return false;
  }
}

type BulkMode = "links" | "search";

export function BulkDownloadTab() {
  const [mode, setMode] = useState<BulkMode>("links");
  const [input, setInput] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [taskIds, setTaskIds] = useState<string[]>([]);
  const cancelled = useRef(false);
  const { toast } = useToast();
  const queue = useQueue();

  const lines = input
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l.length > 0);

  const validCount =
    mode === "links"
      ? lines.filter(isValidUrl).length
      : lines.length; // all non-empty lines are valid queries

  const handleSubmit = async () => {
    if (mode === "links") {
      return handleLinksSubmit();
    }
    return handleSearchSubmit();
  };

  const handleLinksSubmit = async () => {
    const valid = lines.filter(isValidUrl);
    if (valid.length === 0) {
      toast("No valid URLs found", "error");
      return;
    }

    setSubmitting(true);
    setTaskIds([]);
    cancelled.current = false;

    for (let i = 0; i < valid.length; i++) {
      if (cancelled.current) break;
      const link = valid[i];
      try {
        const { task_id } = await DownloadLink(link);
        setTaskIds((prev) => [...prev, task_id]);
        queue.attachTask(task_id);
      } catch (err) {
        toast(`Link ${i + 1} failed: ${err instanceof Error ? err.message : String(err)}`, "error");
      }

      if (i < valid.length - 1) {
        await new Promise((r) => setTimeout(r, 1500));
      }
    }

    setSubmitting(false);
  };

  const handleSearchSubmit = async () => {
    if (lines.length === 0) {
      toast("No queries entered", "error");
      return;
    }

    setSubmitting(true);
    setTaskIds([]);
    cancelled.current = false;

    for (let i = 0; i < lines.length; i++) {
      if (cancelled.current) break;
      const line = lines[i];
      // Parse "Artist - Title" or use the whole line as title.
      let artist = "";
      let title = line;
      const dashIdx = line.indexOf(" - ");
      if (dashIdx > 0) {
        artist = line.slice(0, dashIdx).trim();
        title = line.slice(dashIdx + 3).trim();
      }

      try {
        const data = (await apiSearch(artist, title)) as unknown as SearchResponse;
        if (!data.results || data.results.length === 0) {
          toast(`No results for "${line}"`, "error");
          continue;
        }
        const resp = await apiDownload(data.search_key, 0, artist, title);
        setTaskIds((prev) => [...prev, resp.task_id]);
        queue.attachTask(resp.task_id);
      } catch (err) {
        toast(`"${line}" failed: ${err instanceof Error ? err.message : String(err)}`, "error");
      }

      if (i < lines.length - 1) {
        await new Promise((r) => setTimeout(r, 2000));
      }
    }

    setSubmitting(false);
  };

  // Derive the completed tasks/files from the queue (read-only).
  const completedFiles = useMemo<Map<string, FileInfo[]>>(() => {
    const m = new Map<string, FileInfo[]>();
    for (const tid of taskIds) {
      const t = queue.tasks.find((x) => x.task_id === tid);
      if (t && t.files && t.files.length > 0) {
        m.set(
          tid,
          t.files.map((p) => ({
            filename: p.split(/[\\/]/).pop() ?? p,
            url: p,
            size: 0,
          })),
        );
      }
    }
    return m;
  }, [queue.tasks, taskIds]);

  const pendingTasks = taskIds.filter((tid) => !completedFiles.has(tid));

  const linkCount = lines.filter(isValidUrl).length;

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-lg font-bold text-foreground">Bulk Download</h1>
        <p className="text-sm text-muted-foreground">
          {mode === "links"
            ? "Paste one Spotify/Deezer link per line."
            : "Enter one query per line (\"Artist - Title\" or just a song name)."}
        </p>
      </div>

      <Tabs value={mode} onValueChange={(v) => { setMode(v as BulkMode); setInput(""); setTaskIds([]); }}>
        <TabsList>
          <TabsTrigger value="links">Links</TabsTrigger>
          <TabsTrigger value="search">Search queries</TabsTrigger>
        </TabsList>
        <TabsContent value="links">
          <Card>
            <CardContent className="p-5 space-y-3">
              <textarea
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={`https://open.spotify.com/track/...\nhttps://deezer.com/track/...`}
                rows={8}
                className="w-full px-4 py-3 rounded-lg bg-muted border border-border text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 resize-y font-mono"
              />
              <div className="flex items-center gap-3">
                <Button onClick={handleSubmit} disabled={submitting || validCount === 0}>
                  {submitting ? "Submitting…" : `Download ${validCount} links`}
                </Button>
                <span className="text-xs text-muted-foreground">
                  {linkCount} valid / {lines.length} total
                </span>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="search">
          <Card>
            <CardContent className="p-5 space-y-3">
              <textarea
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={`Aimer - Kataomoi\nYoasobi - Racing Into The Night\nZutomayo - Study Me`}
                rows={8}
                className="w-full px-4 py-3 rounded-lg bg-muted border border-border text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 resize-y font-mono"
              />
              <div className="flex items-center gap-3">
                <Button onClick={handleSubmit} disabled={submitting || validCount === 0}>
                  {submitting ? "Searching…" : `Download ${validCount} queries`}
                </Button>
                <span className="text-xs text-muted-foreground">
                  {validCount} queries
                </span>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {pendingTasks.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            In progress ({pendingTasks.length})
          </p>
          {pendingTasks.map((tid) => {
            const t = queue.tasks.find((x) => x.task_id === tid);
            return (
              <Card key={tid}>
                <CardContent className="p-4 text-sm text-muted-foreground">
                  {t ? (t.query || t.stage || "Working…") : `Waiting for task ${tid.slice(0, 8)}…`}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      {[...completedFiles].map(([tid, files]) => (
        <DownloadFiles key={tid} files={files} />
      ))}
    </div>
  );
}

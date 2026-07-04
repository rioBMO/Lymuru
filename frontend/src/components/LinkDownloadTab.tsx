import { useState, useRef, type FormEvent } from "react";
import { DownloadLink, ChooseLyrics } from "@/lib/api";
import { useTaskProgress } from "@/hooks/useTaskProgress";
import { useToast } from "@/components/Toast";
import { TaskProgress } from "@/components/TaskProgress";
import { LyricsChoice } from "@/components/LyricsChoice";
import { DownloadFiles } from "@/components/DownloadFiles";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export function LinkDownloadTab() {
  const [link, setLink] = useState("");
  const { state, start, reset } = useTaskProgress();
  const { toast } = useToast();
  const running = useRef(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = link.trim();
    if (!trimmed || running.current) return;
    running.current = true;
    reset();
    try {
      const resp = await DownloadLink(trimmed);
      start(resp.task_id);
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Download failed", "error");
    } finally {
      running.current = false;
    }
  }

  async function handleLyricsChoice(choice: "original" | "romanized") {
    if (state.status !== "choosing") return;
    try {
      reset();
      const resp = await ChooseLyrics(state.taskId, choice);
      start(resp.task_id);
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Failed", "error");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="p-5">
          <form onSubmit={handleSubmit} className="flex flex-col sm:flex-row gap-3">
            <Input
              type="url"
              placeholder="Spotify or Deezer link…"
              value={link}
              onChange={(e) => setLink(e.target.value)}
              required
              className="flex-1"
            />
            <Button type="submit" disabled={!link.trim() || running.current}>
              Download
            </Button>
          </form>
        </CardContent>
      </Card>
      {state.status === "running" && <TaskProgress taskId={state.taskId} />}
      {state.status === "choosing" && <LyricsChoice onChoose={handleLyricsChoice} />}
      {state.status === "done" && <DownloadFiles files={state.files} />}
      {state.status === "error" && (
        <div className="bg-destructive/10 text-destructive border border-destructive rounded-xl p-4 text-sm text-center">
          {state.message}
        </div>
      )}
    </div>
  );
}

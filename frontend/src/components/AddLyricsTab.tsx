import { useState, type FormEvent } from "react";
import { AddLyrics } from "@/lib/api";
import { useTaskProgress } from "@/hooks/useTaskProgress";
import { useToast } from "@/components/Toast";
import { FileUpload } from "@/components/FileUpload";
import { TaskProgress } from "@/components/TaskProgress";
import { LyricsChoice } from "@/components/LyricsChoice";
import { DownloadFiles } from "@/components/DownloadFiles";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export function AddLyricsTab() {
  const [file, setFile] = useState<string | null>(null);
  const [artist, setArtist] = useState("");
  const [title, setTitle] = useState("");
  const [showMeta, setShowMeta] = useState(false);
  const { state, start, reset } = useTaskProgress();
  const { toast } = useToast();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!file) return;
    reset();
    try {
      const resp = await AddLyrics(
        file,
        artist.trim(),
        title.trim(),
      );
      start(resp.task_id);
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Failed", "error");
    }
  }

  async function handleLyricsChoice(choice: "original" | "romanized") {
    if (state.status !== "choosing") return;
    try {
      // In Wails mode, lyrics choice is interactive; the user can pick
      // the file from the original/romanized output via the OpenFolder button.
      toast("Lyrics embedded; use the buttons to view the result.", "success");
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Failed", "error");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="p-5">
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <FileUpload id="file-addlyrics" accept=".flac" label="FLAC audio file" onFile={setFile} onClear={() => setFile(null)} />
            <button
              type="button"
              onClick={() => setShowMeta(!showMeta)}
              className="text-xs text-primary hover:underline self-start"
            >
              {showMeta ? "− Hide metadata override" : "+ Override metadata (optional)"}
            </button>
            {showMeta && (
              <div className="flex gap-3">
                <Input
                  placeholder="Artist"
                  value={artist}
                  onChange={(e) => setArtist(e.target.value)}
                  className="flex-1"
                />
                <Input
                  placeholder="Title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="flex-1"
                />
              </div>
            )}
            <Button type="submit" disabled={!file}>
              Add Lyrics
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

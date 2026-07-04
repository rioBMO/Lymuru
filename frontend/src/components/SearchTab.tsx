import { useState, useRef, type FormEvent } from "react";
import { Search, Download, ChooseLyrics } from "@/lib/api";
import type { SearchResponse, SearchResult } from "@/lib/api";
import { useTaskProgress } from "@/hooks/useTaskProgress";
import { TaskProgress } from "@/components/TaskProgress";
import { LyricsChoice } from "@/components/LyricsChoice";
import { DownloadFiles } from "@/components/DownloadFiles";
import { useToast } from "@/components/Toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";

export function SearchTab() {
  const [artist, setArtist] = useState("");
  const [title, setTitle] = useState("");
  const [searchKey, setSearchKey] = useState("");
  const [results, setResults] = useState<SearchResult[] | null>(null);
  const [searching, setSearching] = useState(false);
  const fetchingChoice = useRef(false);
  const { state, start: startProgress, reset } = useTaskProgress();
  const { toast } = useToast();

  async function handleSearch(e: FormEvent) {
    e.preventDefault();
    const a = artist.trim();
    const t = title.trim();
    if (!a && !t) return; // at least one field required

    setSearching(true);
    setResults(null);
    reset();
    try {
      const data = (await Search(a, t)) as unknown as SearchResponse;
      setSearchKey(data.search_key);
      setResults(data.results);
      if (!data.results || data.results.length === 0) {
        toast("No results found", "error");
      }
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Search failed", "error");
    } finally {
      setSearching(false);
    }
  }

  async function handleDownload(index: number) {
    if (fetchingChoice.current) return;
    fetchingChoice.current = true;
    reset();
    try {
      const resp = await Download(searchKey, index, artist.trim(), title.trim());
      startProgress(resp.task_id);
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Download failed", "error");
    } finally {
      fetchingChoice.current = false;
    }
  }

  async function handleLyricsChoice(choice: "original" | "romanized") {
    if (state.status !== "choosing") return;
    try {
      reset();
      const resp = await ChooseLyrics(state.taskId, choice);
      startProgress(resp.task_id);
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Lyrics choice failed", "error");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="p-5">
          <form onSubmit={handleSearch} className="flex flex-col sm:flex-row gap-3">
            <Input
              placeholder="Artist (optional)"
              value={artist}
              onChange={(e) => setArtist(e.target.value)}
              className="flex-1"
            />
            <Input
              placeholder="Title (optional)"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="flex-1"
            />
            <Button type="submit" disabled={searching} className="sm:w-auto">
              {searching ? <Spinner size="sm" className="text-primary-foreground" /> : null}
              {searching ? "Searching…" : "Search"}
            </Button>
          </form>
        </CardContent>
      </Card>

      {searching && (
        <div className="flex flex-col items-center gap-2 py-6">
          <img src="/assets/image/lymuru-mascot.png" className="h-14 w-auto animate-bounce" alt="" />
          <p className="text-sm text-muted-foreground">Searching…</p>
        </div>
      )}

      {results !== null && results.length === 0 && !searching && (
        <div className="flex flex-col items-center gap-2 py-6">
          <img src="/assets/image/lymuru-not-found.png" className="h-14 w-auto" alt="" />
          <p className="text-sm text-muted-foreground">No results found</p>
        </div>
      )}

      {results !== null && results.length > 0 && (
        <div className="flex flex-col gap-2">
          {results.map((r) => (
            <Card key={r.index}>
              <CardContent className="p-4 flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-foreground truncate">{r.title}</div>
                  {r.description && (
                    <div className="text-xs text-muted-foreground truncate">{r.description}</div>
                  )}
                </div>
                <Button onClick={() => handleDownload(r.index)} size="sm">
                  Download
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

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

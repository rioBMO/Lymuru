import { useState, type FormEvent } from "react";
import { AddLyrics } from "@/lib/api";
import { useToast } from "@/components/Toast";
import { FileUpload } from "@/components/FileUpload";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export function AddLyricsTab() {
  const [file, setFile] = useState<string | null>(null);
  const [artist, setArtist] = useState("");
  const [title, setTitle] = useState("");
  const [showMeta, setShowMeta] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { toast } = useToast();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!file) return;
    setIsSubmitting(true);
    try {
      const resp = await AddLyrics(
        file,
        artist.trim(),
        title.trim(),
      );
      toast(resp, "success");
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setIsSubmitting(false);
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
            <Button type="submit" disabled={!file || isSubmitting}>
              {isSubmitting ? "Adding Lyrics..." : "Add Lyrics"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

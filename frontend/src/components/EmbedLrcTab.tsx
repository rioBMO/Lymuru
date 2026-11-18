import { useState, type FormEvent } from "react";
import { EmbedLrc } from "@/lib/api";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import { FileUpload } from "@/components/FileUpload";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export function EmbedLrcTab() {
  const [flac, setFlac] = useState<string | null>(null);
  const [lrc, setLrc] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!flac || !lrc || loading) return;
    setLoading(true);
    try {
      await EmbedLrc(flac, lrc);
      toast.success("Lyrics embedded");
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Embed failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card>
      <CardContent className="p-5">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <FileUpload id="file-embed-flac" accept=".flac,.mp3,.m4a" label="Audio file" onFile={setFlac} onClear={() => setFlac(null)} />
          <FileUpload id="file-embed-lrc" accept=".lrc,.txt" label="LRC lyrics file" onFile={setLrc} onClear={() => setLrc(null)} />
          <Button type="submit" disabled={!flac || !lrc || loading}>
            {loading ? "Processing…" : "Embed LRC"}
          </Button>
          {loading && (
            <div className="flex justify-center">
              <img src="/assets/image/lymuru-mascot.png" className="h-12 w-auto animate-bounce" alt="" />
            </div>
          )}
        </form>
      </CardContent>
    </Card>
  );
}

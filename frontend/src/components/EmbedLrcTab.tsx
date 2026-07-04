import { useState, type FormEvent } from "react";
import { EmbedLrc } from "@/lib/api";
import { useToast } from "@/components/Toast";
import { FileUpload } from "@/components/FileUpload";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export function EmbedLrcTab() {
  const [flac, setFlac] = useState<string | null>(null);
  const [lrc, setLrc] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { toast } = useToast();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!flac || !lrc || loading) return;
    setLoading(true);
    try {
      await EmbedLrc(flac, lrc);
      toast("FLAC processed with embedded lyrics", "success");
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Embed failed", "error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card>
      <CardContent className="p-5">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <FileUpload id="file-embed-flac" accept=".flac" label="FLAC audio file" onFile={setFlac} onClear={() => setFlac(null)} />
          <FileUpload id="file-embed-lrc" accept=".lrc,.txt" label="LRC lyrics file" onFile={setLrc} onClear={() => setLrc(null)} />
          <Button type="submit" disabled={!flac || !lrc || loading}>
            {loading ? "Processing…" : "Embed LRC into FLAC"}
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

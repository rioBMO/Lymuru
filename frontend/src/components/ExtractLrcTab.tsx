import { useState, type FormEvent } from "react";
import { ExtractLrc, OpenFolder, type ExtractResult } from "@/lib/api";
import { useToast } from "@/components/Toast";
import { FileUpload } from "@/components/FileUpload";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function ExtractLrcTab() {
  const [file, setFile] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ lyrics: string; isSynced: boolean; url: string } | null>(null);
  const { toast } = useToast();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!file || loading) return;
    setLoading(true);
    setResult(null);
    try {
      const data = (await ExtractLrc(file)) as ExtractResult;
      setResult({ lyrics: data.lyrics, isSynced: data.is_synced, url: data.output_url });
    } catch (err: unknown) {
      toast(err instanceof Error ? err.message : "Extraction failed", "error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="p-5">
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <FileUpload id="file-extract" accept=".flac" label="FLAC audio file" onFile={setFile} onClear={() => { setFile(null); setResult(null); }} />
            <Button type="submit" disabled={!file || loading}>
              {loading ? "Extracting…" : "Extract Lyrics"}
            </Button>
          </form>
        </CardContent>
      </Card>

      {loading && (
        <div className="flex justify-center">
          <img src="/assets/image/lymuru-mascot.png" className="h-12 w-auto animate-bounce" alt="" />
        </div>
      )}

      {result && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">
              Extracted lyrics {result.isSynced ? "(synced LRC)" : "(plain text)"}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-xs text-muted-foreground whitespace-pre-wrap max-h-64 overflow-auto bg-muted rounded-lg p-3 border border-border">
              {result.lyrics}
            </pre>
            <Button onClick={() => OpenFolder(result.url)} className="mt-3" variant="default">
              Open in folder
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

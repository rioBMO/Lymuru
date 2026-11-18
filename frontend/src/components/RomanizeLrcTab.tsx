import { useState, type FormEvent } from "react";
import { RomanizeLrc, OpenFolder } from "@/lib/api";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import { FileUpload } from "@/components/FileUpload";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function RomanizeLrcTab() {
  const [file, setFile] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ text: string | null; url: string | null; message: string } | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!file || loading) return;
    setLoading(true);
    setResult(null);
    try {
      const data = await RomanizeLrc(file);
      setResult({ text: data.romanized, url: data.download_url, message: data.message });
      if (!data.romanized) toast.info(data.message);
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Romanization failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="p-5">
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <FileUpload id="file-romanize" accept=".lrc,.txt" label="LRC lyrics file" onFile={setFile} onClear={() => { setFile(null); setResult(null); }} />
            <Button type="submit" disabled={!file || loading}>
              {loading ? "Romanizing…" : "Romanize LRC"}
            </Button>
          </form>
        </CardContent>
      </Card>

      {loading && (
        <div className="flex justify-center">
          <img src="/assets/image/lymuru-mascot.png" className="h-12 w-auto animate-bounce" alt="" />
        </div>
      )}

      {result?.text && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Romanized lyrics preview</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-xs text-muted-foreground whitespace-pre-wrap max-h-64 overflow-auto bg-muted rounded-lg p-3 border border-border">
              {result.text}
            </pre>
            {result.url && (
              <Button onClick={async () => {
                try {
                  await OpenFolder(result.url!);
                } catch (error) {
                  toast.error(`Error opening folder: ${error}`);
                }
              }} className="mt-3" variant="default">
                Open in folder
              </Button>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

import { useState, type DragEvent } from "react";
import { X, FileAudio, FileText } from "lucide-react";
import { PickFile } from "@/lib/api";
import { Button } from "@/components/ui/button";

interface Props {
  id: string;
  accept: string;
  label: string;
  onFile: (filePath: string) => void;
  onClear?: () => void;
  disabled?: boolean;
}

/**
 * FileUpload in the Wails version opens the native file dialog
 * via the `PickFile` binding (a Go-side call to
 * `runtime.OpenFileDialog`). The returned path is forwarded to the
 * caller as both a path and a basename.
 */
export function FileUpload({ id, accept, label, onFile, onClear, disabled }: Props) {
  const [filePath, setFilePath] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);

  async function handlePick() {
    if (disabled) return;
    try {
      // The Go side picks filter description and pattern from the accept
      // string (e.g. ".flac,.lrc" -> description "Audio files",
      // pattern "*.flac;*.lrc").
      const exts = accept
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean)
        .map((s) => (s.startsWith(".") ? "*" + s : s));
      const pattern = exts.join(";");
      const desc =
        accept.includes(".flac") && accept.includes(".lrc")
          ? "Audio or lyrics files"
          : accept.includes(".flac")
            ? "FLAC audio"
            : "LRC lyrics";
      const path = await PickFile(desc, pattern);
      if (!path) return;
      setFilePath(path);
      onFile(path);
    } catch (err) {
      console.error("PickFile failed:", err);
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    setDragOver(false);
    // Native drag-and-drop into a Wails WebView: the file path is
    // exposed via the `path` property of the File object in newer
    // Wails versions. Otherwise, open the dialog.
    const f = e.dataTransfer.files?.[0] as (File & { path?: string }) | undefined;
    if (f && f.path) {
      setFilePath(f.path);
      onFile(f.path);
    } else {
      handlePick();
    }
  }

  function handleClear() {
    setFilePath(null);
    onClear?.();
  }

  const fileName = filePath ? filePath.split(/[\\/]/).pop() ?? filePath : null;
  const isAudio = accept.includes(".flac");

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-xs font-semibold text-muted-foreground">
        {label}
      </label>

      {fileName ? (
        <div className="flex items-center gap-2 px-3 py-2.5 rounded-lg border border-border bg-muted text-sm text-foreground">
          {isAudio ? (
            <FileAudio className="h-4 w-4 text-muted-foreground" />
          ) : (
            <FileText className="h-4 w-4 text-muted-foreground" />
          )}
          <span className="flex-1 truncate" title={filePath ?? ""}>
            {fileName}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={handleClear}
            title="Remove file"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      ) : (
        <div
          onDrop={handleDrop}
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onClick={handlePick}
          className={`px-3 py-4 rounded-lg border-2 border-dashed text-center cursor-pointer transition-colors text-sm
            ${dragOver ? "border-primary bg-primary/5" : "border-border hover:border-primary/40 text-muted-foreground"}
            ${disabled ? "opacity-50 cursor-not-allowed" : ""}`}
        >
          <p>Click to browse or drop a file here</p>
          <p className="text-xs mt-1 text-muted-foreground/70">
            {accept || "any file"}
          </p>
        </div>
      )}

      <input id={id} type="hidden" value={filePath ?? ""} readOnly />
    </div>
  );
}

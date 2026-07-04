import { FolderOpen, Download } from "lucide-react";
import { formatSize } from "@/lib/format";
import { OpenFolder } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export interface FileInfo {
  filename: string;
  url: string;     // absolute path
  size: number;
}

interface Props {
  files: FileInfo[];
}

function fileDir(p: string): string {
  const sep = p.includes("\\") ? "\\" : "/";
  const idx = p.lastIndexOf(sep);
  return idx === -1 ? p : p.substring(0, idx);
}

export function DownloadFiles({ files }: Props) {
  if (files.length === 0) return null;

  // Use the directory of the first file as the "open in folder" target.
  const firstDir = fileDir(files[0].url);

  return (
    <Card>
      <CardContent className="p-5 space-y-3">
        <div className="flex items-center gap-2">
          <img
            src="/assets/image/lymuru-found.png"
            alt="Complete"
            className="h-10 w-auto"
          />
          <span className="text-sm font-semibold text-foreground">Download ready</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {files.map((f) => (
            <Button
              key={f.filename}
              asChild
              size="sm"
              variant="default"
            >
              <a
                href={`file://${f.url.replace(/\\/g, "/")}`}
                download={f.filename}
                title={`Open ${f.filename}`}
              >
                <Download />
                {f.filename}{f.size > 0 ? ` (${formatSize(f.size)})` : ""}
              </a>
            </Button>
          ))}
        </div>
        {firstDir && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => OpenFolder(firstDir)}
            className="gap-1.5"
          >
            <FolderOpen className="h-3.5 w-3.5" />
            Open downloads folder
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

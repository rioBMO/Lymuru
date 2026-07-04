import { FolderOpen } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";

export function FileManagerPage() {
  return (
    <div className="max-w-2xl">
      <Card>
        <CardContent className="p-10 flex flex-col items-center gap-3 text-center">
          <img
            src="/assets/image/lymuru-mascot.png"
            alt=""
            className="h-20 w-auto opacity-40"
          />
          <FolderOpen className="h-8 w-8 text-muted-foreground" />
          <div>
            <h2 className="text-base font-semibold text-foreground">File Manager</h2>
            <p className="text-sm text-muted-foreground mt-1">
              Browse and manage downloaded files. Coming soon.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

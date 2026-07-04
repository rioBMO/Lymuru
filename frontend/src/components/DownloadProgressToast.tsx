import { ListMusic } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";

interface Props {
  activeCount: number;
  currentStage?: string;
  currentPercent?: number;
  onClick: () => void;
}

export function DownloadProgressToast({
  activeCount,
  currentStage,
  currentPercent,
  onClick,
}: Props) {
  if (activeCount === 0) return null;

  const pct = Math.max(0, Math.min(100, Math.round(currentPercent ?? 0)));
  const isPreparing = pct === 0;

  return (
    <div className="fixed bottom-6 right-6 z-40 w-72">
      <Card className="shadow-lg cursor-pointer hover:shadow-xl transition-shadow" onClick={onClick}>
        <CardContent className="p-3">
          <div className="flex items-center gap-2 mb-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
              {isPreparing ? <Spinner size="sm" className="text-primary-foreground" /> : <ListMusic className="h-3.5 w-3.5" />}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-xs font-semibold text-foreground truncate">
                {currentStage ?? "Working…"}
              </p>
              <p className="text-[10px] text-muted-foreground">
                {activeCount} active task{activeCount === 1 ? "" : "s"}
              </p>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onClick();
              }}
              className="h-7 px-2 text-[10px]"
            >
              View
            </Button>
          </div>
          <div className="h-1.5 rounded-full bg-muted overflow-hidden">
            <div
              className={cn(
                "h-full rounded-full bg-primary transition-all duration-300",
                isPreparing && "animate-pulse w-1/3",
              )}
              style={isPreparing ? undefined : { width: `${pct}%` }}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

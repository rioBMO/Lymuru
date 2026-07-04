import { Minus, Square, X, Maximize2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
} from "../../wailsjs/runtime/runtime";

/**
 * Custom frameless title bar. Provides window controls (minimize,
 * maximize/restore, close) and a draggable region.
 *
 * The `--wails-draggable` CSS class is set by Wails to enable
 * native window dragging on this element.
 */
export function TitleBar() {
  return (
    <div
      data-wails-draggable
      className="flex h-8 shrink-0 items-center justify-between border-b border-border bg-card/80 backdrop-blur select-none"
    >
      <div className="flex items-center gap-2 pl-3">
        <img
          src="/assets/image/lymuru-logo.png"
          alt=""
          className="h-4 w-4 object-contain"
        />
        <span className="text-xs font-semibold text-foreground">
          Lymuru
        </span>
      </div>
      <div className="flex h-full" data-wails-no-drag>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-10 rounded-none hover:bg-accent"
          onClick={() => WindowMinimise()}
          aria-label="Minimize"
        >
          <Minus className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-10 rounded-none hover:bg-accent"
          onClick={() => WindowToggleMaximise()}
          aria-label="Maximize"
        >
          <Maximize2 className="h-3 w-3" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-10 rounded-none hover:bg-destructive hover:text-destructive-foreground"
          onClick={() => Quit()}
          aria-label="Close"
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}

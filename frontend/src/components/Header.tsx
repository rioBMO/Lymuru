import { Menu, Moon, Sun } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

interface Props {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
  onOpenMobileMenu?: () => void;
  version?: string;
}

export function Header({
  themeMode,
  onToggleTheme,
  onOpenMobileMenu,
  version,
}: Props) {

  return (
    <header className="shrink-0 z-30 flex h-14 items-center gap-3 border-b border-border bg-card/80 px-4 md:px-6 backdrop-blur">
      {onOpenMobileMenu && (
        <Button
          variant="ghost"
          size="icon"
          className="md:hidden"
          onClick={onOpenMobileMenu}
          aria-label="Open menu"
        >
          <Menu className="h-5 w-5" />
        </Button>
      )}
      <div className="flex items-center gap-2">
        <img
          src="/assets/image/lymuru-logo.png"
          alt=""
          className="h-7 w-7 object-contain md:hidden"
        />
        <h1 className="text-base font-semibold text-foreground">Lymuru</h1>
        {version && version !== "0.0.0" && (
          <Badge variant="outline" className="hidden sm:inline-flex text-[10px]">
            v{version}
          </Badge>
        )}
      </div>
      <div className="ml-auto flex items-center gap-2">

        <Button
          variant="ghost"
          size="icon"
          onClick={onToggleTheme}
          aria-label="Toggle theme"
        >
          {themeMode === "dark" ? (
            <Sun className="h-4 w-4" />
          ) : (
            <Moon className="h-4 w-4" />
          )}
        </Button>
      </div>
    </header>
  );
}

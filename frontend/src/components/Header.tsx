import { Menu, Moon, Sun, Circle, CircleDot, CircleDashed, AlertCircle, ShieldAlert, RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface SidecarStatus {
  status: string;
  message: string;
}

interface Props {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
  onOpenMobileMenu?: () => void;
  sidecar?: SidecarStatus;
  version?: string;
  onRestartSidecar?: () => void;
  onOpenAuth?: () => void;
}

const STATUS_LABEL: Record<string, { label: string; icon: typeof Circle; color: string }> = {
  online:        { label: "Sidecar online",   icon: CircleDot,    color: "text-emerald-500" },
  starting:      { label: "Sidecar starting", icon: CircleDashed, color: "text-muted-foreground" },
  stopped:       { label: "Sidecar stopped",  icon: Circle,       color: "text-muted-foreground" },
  auth_required: { label: "Auth required",    icon: ShieldAlert,  color: "text-amber-500" },
  error:         { label: "Sidecar error",    icon: AlertCircle,  color: "text-destructive" },
};

export function Header({
  themeMode,
  onToggleTheme,
  onOpenMobileMenu,
  sidecar,
  version,
  onRestartSidecar,
  onOpenAuth,
}: Props) {
  const statusKey = sidecar?.status ?? "starting";
  const statusInfo = STATUS_LABEL[statusKey] ?? STATUS_LABEL.starting;
  const StatusIcon = statusInfo.icon;
  const showRestart = Boolean(onRestartSidecar) && (statusKey === "error" || statusKey === "stopped");
  const isAuthRequired = statusKey === "auth_required";

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
        {sidecar && (
          isAuthRequired && onOpenAuth ? (
            <Button
              variant="ghost"
              size="sm"
              className={cn("flex items-center gap-1.5 text-xs h-auto py-1", statusInfo.color)}
              onClick={onOpenAuth}
              title={"Click to enter verification code — " + (sidecar.message || "")}
            >
              <StatusIcon className="h-3.5 w-3.5 shrink-0" />
              <span className="hidden truncate sm:inline">
                Auth required — click to enter code
              </span>
            </Button>
          ) : (
            <div
              className={cn(
                "flex max-w-[60vw] items-center gap-1.5 text-xs",
                statusInfo.color,
              )}
              title={sidecar.message || statusInfo.label}
            >
              <StatusIcon className="h-3.5 w-3.5 shrink-0" />
              <span className="hidden truncate sm:inline">
                {sidecar.message ? `${statusInfo.label}: ${sidecar.message}` : statusInfo.label}
              </span>
            </div>
          )
        )}
        {showRestart && onRestartSidecar && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onRestartSidecar}
            aria-label="Restart sidecar"
            title="Restart sidecar"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        )}
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

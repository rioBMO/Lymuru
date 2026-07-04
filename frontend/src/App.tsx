import { useState, useCallback, useEffect } from "react";
import { initTheme, applyTheme } from "@/lib/themes";
import { GetAuthState, GetVersion, GetSidecarInfo, RestartSidecar, Events, type SidecarInfo } from "@/lib/api";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import { useQueue, QueueProvider } from "@/context/QueueContext";
import { ToastProvider } from "@/components/Toast";
import { Sidebar, type Page } from "@/components/Sidebar";
import { Header } from "@/components/Header";
import { TitleBar } from "@/components/TitleBar";
import { DownloadProgressToast } from "@/components/DownloadProgressToast";
import { DownloadQueue } from "@/components/DownloadQueue";
import { CooldownBanner } from "@/components/CooldownBanner";
import { AuthDialog } from "@/components/AuthDialog";
import { TooltipProvider } from "@/components/ui/tooltip";
import { HomePage } from "@/components/pages/HomePage";
import { LyricsManagerPage } from "@/components/pages/LyricsManagerPage";
import { HistoryPage } from "@/components/pages/HistoryPage";
import { FileManagerPage } from "@/components/pages/FileManagerPage";
import { SettingsPage } from "@/components/pages/SettingsPage";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  House,
  FileText,
  History as HistoryIcon,
  FolderOpen,
  Settings as SettingsIcon,
} from "lucide-react";

function PageRouter({
  page,
  themeMode,
  onToggleTheme,
}: {
  page: Page;
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
}) {
  switch (page) {
    case "home":
      return <HomePage />;
    case "lyrics":
      return <LyricsManagerPage />;
    case "history":
      return <HistoryPage />;
    case "files":
      return <FileManagerPage />;
    case "settings":
      return (
        <SettingsPage themeMode={themeMode} onToggleTheme={onToggleTheme} />
      );
    default:
      return <HomePage />;
  }
}

const MOBILE_NAV_ITEMS: { id: Page; label: string; icon: typeof House }[] = [
  { id: "home", label: "Home", icon: House },
  { id: "lyrics", label: "Lyrics Manager", icon: FileText },
  { id: "history", label: "History", icon: HistoryIcon },
  { id: "files", label: "File Manager", icon: FolderOpen },
  { id: "settings", label: "Settings", icon: SettingsIcon },
];

function MobileNav({
  open,
  onOpenChange,
  currentPage,
  onPageChange,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  currentPage: Page;
  onPageChange: (p: Page) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xs">
        <DialogHeader>
          <DialogTitle>Navigation</DialogTitle>
          <DialogDescription>Choose a page to navigate to.</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-1">
          {MOBILE_NAV_ITEMS.map((it) => {
            const Icon = it.icon;
            return (
              <Button
                key={it.id}
                variant={currentPage === it.id ? "secondary" : "ghost"}
                className="justify-start gap-2"
                onClick={() => {
                  onPageChange(it.id);
                  onOpenChange(false);
                }}
              >
                <Icon className="h-4 w-4" />
                {it.label}
              </Button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}

interface SidecarStatus {
  status: string;
  message: string;
}

function AuthenticatedApp({
  themeMode,
  onToggleTheme,
}: {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
}) {
  const [page, setPage] = useState<Page>("home");
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [sidecar, setSidecar] = useState<SidecarStatus>({ status: "starting", message: "" });
  const [authRequired, setAuthRequired] = useState(false);
  const [authPhone, setAuthPhone] = useState("");
  const [version, setVersion] = useState<string>("0.0.0");
  const queue = useQueue();

  // Poll sidecar status on mount and via Wails events.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [v, info, a] = await Promise.all([
          GetVersion(),
          GetSidecarInfo(),
          GetAuthState(),
        ]);
        if (cancelled) return;
        setVersion(typeof v === "string" ? v : "0.0.0");
        const si = info as SidecarInfo;
        setSidecar({ status: si.status || "starting", message: si.message || "" });
        const authState = a as { state?: string } | undefined;
        if (authState && authState.state === "auth_required") {
          setAuthRequired(true);
          // Phone comes from the durable sidecar status message, not GetAuthState.
          if (si.message) setAuthPhone(String(si.message));
        }
      } catch {
        /* ignore — Wails bindings may not be ready yet */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const onStatus = (...args: unknown[]) => {
      const data = args[0] as SidecarStatus | undefined;
      if (!data) return;
      setSidecar({ status: String(data.status), message: String(data.message ?? "") });
      if (data.status === "auth_required") {
        setAuthRequired(true);
        if (data.message) setAuthPhone(String(data.message));
      }
      if (data.status === "online") setAuthRequired(false);
    };
    EventsOn(Events.Sidecar, onStatus as (...args: unknown[]) => void);
    return () => {
      EventsOff(Events.Sidecar);
    };
  }, []);

  const topTask = queue.tasks
    .filter((t) => t.phase !== "complete" && !t.error)
    .sort((a, b) => (b.download_percent || 0) - (a.download_percent || 0))[0];

  const handleRestartSidecar = useCallback(async () => {
    try {
      await RestartSidecar();
    } catch (err) {
      console.error("Failed to restart sidecar:", err);
    }
  }, []);

  return (
    <TooltipProvider>
      <div className="h-screen flex flex-col overflow-hidden bg-background text-foreground">
        <TitleBar />
        <div className="flex-1 flex overflow-hidden">
          <Sidebar currentPage={page} onPageChange={setPage} />
          <div className="md:ml-14 flex-1 flex flex-col overflow-hidden">
            <Header
              themeMode={themeMode}
              onToggleTheme={onToggleTheme}
              onOpenMobileMenu={() => setMobileNavOpen(true)}
              sidecar={sidecar}
              version={version}
              onRestartSidecar={handleRestartSidecar}
              onOpenAuth={() => setAuthRequired(true)}
            />
            <CooldownBanner message={undefined} />
            <main className="flex-1 overflow-y-auto">
              <div className="max-w-4xl mx-auto p-4 md:p-8">
                <PageRouter
                  page={page}
                  themeMode={themeMode}
                  onToggleTheme={onToggleTheme}
                />
              </div>
            </main>
          </div>
        </div>
        <MobileNav
          open={mobileNavOpen}
          onOpenChange={setMobileNavOpen}
          currentPage={page}
          onPageChange={setPage}
        />
        <DownloadProgressToast
          activeCount={queue.activeCount}
          currentStage={topTask?.stage}
          currentPercent={topTask?.download_percent}
          onClick={queue.openQueue}
        />
        <DownloadQueue
          open={queue.isQueueOpen}
          tasks={queue.tasks}
          onClose={queue.closeQueue}
          onCancel={queue.cancel}
          onClearAll={queue.clearAll}
        />
      </div>
      <AuthDialog
        open={authRequired}
        phone={authPhone}
        onAuthenticated={() => setAuthRequired(false)}
        onClose={() => setAuthRequired(false)}
      />
    </TooltipProvider>
  );
}

export default function App() {
  const [themeMode, setThemeMode] = useState<"light" | "dark">(() => {
    try {
      const saved = localStorage.getItem("lymuru_theme");
      return saved === "dark" ? "dark" : "light";
    } catch {
      return "light";
    }
  });

  // Apply theme on mount.
  useEffect(() => {
    initTheme();
    try {
      const saved = localStorage.getItem("lymuru_theme");
      if (saved === "light" || saved === "dark") setThemeMode(saved);
    } catch {
      /* noop */
    }
  }, []);

  const toggleTheme = useCallback(() => {
    const next = themeMode === "light" ? "dark" : "light";
    applyTheme(next);
    setThemeMode(next);
  }, [themeMode]);

  return (
    <ToastProvider>
      <QueueProvider>
        <AuthenticatedApp
          themeMode={themeMode}
          onToggleTheme={toggleTheme}
        />
      </QueueProvider>
    </ToastProvider>
  );
}

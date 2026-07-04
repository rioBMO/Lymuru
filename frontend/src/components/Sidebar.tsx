import {
  House,
  FileText,
  History,
  FolderOpen,
  Settings as SettingsIcon,
} from "lucide-react";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export type Page = "home" | "lyrics" | "history" | "files" | "settings";

interface NavItem {
  id: Page;
  label: string;
  icon: typeof House;
}

const NAV_ITEMS: NavItem[] = [
  { id: "home", label: "Home", icon: House },
  { id: "lyrics", label: "Lyrics Manager", icon: FileText },
  { id: "history", label: "History", icon: History },
  { id: "files", label: "File Manager", icon: FolderOpen },
  { id: "settings", label: "Settings", icon: SettingsIcon },
];

interface Props {
  currentPage: Page;
  onPageChange: (page: Page) => void;
}

export function Sidebar({ currentPage, onPageChange }: Props) {
  return (
    <aside className="fixed top-0 left-0 z-40 hidden md:flex h-screen w-14 flex-col items-center gap-1 border-r border-border bg-card py-3">
      {/* Logo */}
      <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-md overflow-hidden">
        <img
          src="/assets/image/lymuru-logo.png"
          alt="Lymuru"
          className="h-9 w-9 object-contain"
        />
      </div>

      <nav className="flex flex-1 flex-col items-center gap-1">
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          const isActive = currentPage === item.id;
          return (
            <Tooltip key={item.id} delayDuration={0}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  onClick={() => onPageChange(item.id)}
                  aria-label={item.label}
                  aria-current={isActive ? "page" : undefined}
                  className={cn(
                    "flex h-9 w-9 items-center justify-center rounded-md transition-colors cursor-pointer",
                    isActive
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  )}
                >
                  <Icon className="h-4 w-4" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">{item.label}</TooltipContent>
            </Tooltip>
          );
        })}
      </nav>
    </aside>
  );
}

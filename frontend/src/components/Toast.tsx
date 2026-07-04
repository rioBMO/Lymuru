import { useEffect, useState, useCallback, createContext, useContext, type ReactNode } from "react";
import { cn } from "@/lib/utils";

interface Toast {
  id: number;
  message: string;
  variant: "error" | "success" | "";
}

interface ToastCtx {
  toast: (message: string, variant?: "error" | "success") => void;
}

const Ctx = createContext<ToastCtx>({ toast: () => {} });
let _nextId = 0;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Toast[]>([]);

  const toast = useCallback((message: string, variant: "error" | "success" | "" = "") => {
    const id = ++_nextId;
    setItems((prev) => [...prev, { id, message, variant }]);
    setTimeout(() => {
      setItems((prev) => prev.filter((t) => t.id !== id));
    }, 3500);
  }, []);

  return (
    <Ctx.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-20 right-4 z-50 flex flex-col gap-2 max-w-sm">
        {items.map((t) => (
          <div
            key={t.id}
            className={cn(
              "animate-[fadeIn_0.3s_ease] px-4 py-3 rounded-lg shadow-md text-sm font-medium border",
              t.variant === "error"
                ? "bg-destructive/10 text-destructive border-destructive"
                : t.variant === "success"
                ? "bg-emerald-500/15 text-emerald-600 border-emerald-500"
                : "bg-card text-foreground border-border"
            )}
          >
            {t.message}
          </div>
        ))}
      </div>
    </Ctx.Provider>
  );
}

export function useToast() {
  return useContext(Ctx);
}

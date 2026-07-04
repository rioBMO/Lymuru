import { Info } from "lucide-react";

interface Props {
  message?: string;
}

/**
 * Placeholder banner for future backend cooldown / rate-limit messaging.
 * Currently hidden unless a message is provided.
 */
export function CooldownBanner({ message }: Props) {
  if (!message) return null;

  return (
    <div className="flex items-center gap-2 px-4 py-2 text-xs border-b border-border bg-muted text-muted-foreground">
      <Info className="h-3.5 w-3.5" />
      <span>{message}</span>
    </div>
  );
}

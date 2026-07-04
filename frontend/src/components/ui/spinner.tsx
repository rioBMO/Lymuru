import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface SpinnerProps {
  className?: string;
  size?: "sm" | "md" | "lg";
}

function Spinner({ className, size = "md" }: SpinnerProps) {
  const sizeClass = {
    sm: "h-4 w-4",
    md: "h-5 w-5",
    lg: "h-8 w-8",
  }[size];

  return (
    <Loader2
      data-slot="spinner"
      className={cn("animate-spin text-muted-foreground", sizeClass, className)}
    />
  );
}

export { Spinner };

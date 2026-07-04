import { useState } from "react";
import { ShieldAlert, X } from "lucide-react";
import { SubmitAuthCode, GetAuthState } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/Toast";

interface Props {
  open: boolean;
  phone?: string;
  onAuthenticated: () => void;
  onClose?: () => void;
}

/**
 * Modal dialog shown when the Telegram sidecar reports `auth_needed`.
 * The user types the verification code sent to their phone; we forward
 * it to the Go backend via `SubmitAuthCode`.
 */
export function AuthDialog({ open, phone, onAuthenticated, onClose }: Props) {
  const [code, setCode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { toast } = useToast();

  if (!open) return null;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!code.trim()) return;
    setSubmitting(true);
    try {
      await SubmitAuthCode(code.trim());
      toast("Verification code submitted", "success");
      setCode("");
      // Poll GetAuthState for up to 12 seconds waiting for auth to complete.
      const started = Date.now();
      while (Date.now() - started < 12000) {
        await new Promise((r) => setTimeout(r, 1500));
        try {
          const result = (await GetAuthState()) as { state?: string };
          if (result?.state === "authenticated") {
            onAuthenticated();
            return;
          }
          if (result?.state === "error") break;
        } catch {
          /* keep polling */
        }
      }
      // If we get here, auth didn't succeed within the timeout.
      // The status event will eventually update the UI.
      setSubmitting(false);
    } catch (err: unknown) {
      setSubmitting(false);
      setError(err instanceof Error ? err.message : "Failed to submit code");
    }
  }

  const phoneLabel = phone
    ? phone.replace(/^(\+\d+)/, "$1 ") // e.g. "+6281384273419" → "+6281384273419 "
    : "your phone";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <Card className="w-full max-w-sm mx-4 rounded-2xl shadow-lg">
        <CardHeader>
          <div className="flex items-start justify-between gap-2">
            <div className="flex items-center gap-2">
              <ShieldAlert className="h-5 w-5 text-amber-500" />
              <CardTitle>Telegram Sign-in Required</CardTitle>
            </div>
            {onClose && (
              <Button
                variant="ghost"
                size="icon"
                onClick={onClose}
                aria-label="Close"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>
          <CardDescription>
            A verification code was sent to{" "}
            <span className="font-mono font-semibold">{phoneLabel}</span>.
            Open Telegram and enter the code below.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="auth-code">Verification code</Label>
              <Input
                id="auth-code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="e.g. 12345"
                autoComplete="one-time-code"
                disabled={submitting}
                className="font-mono tracking-widest text-center"
              />
            </div>
            {error && (
              <p className="text-xs text-destructive text-center">{error}</p>
            )}
            <Button type="submit" disabled={!code.trim() || submitting} className="w-full">
              {submitting ? "Submitting…" : "Submit code"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

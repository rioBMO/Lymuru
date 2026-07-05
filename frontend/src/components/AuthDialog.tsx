import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";

interface AuthDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSubmitCode: (code: string) => Promise<void>;
}

export function AuthDialog({ open, onOpenChange, onSubmitCode }: AuthDialogProps) {
    const [code, setCode] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");

    const handleSubmit = async () => {
        if (!code.trim()) return;
        setSubmitting(true);
        setError("");
        try {
            await onSubmitCode(code.trim());
            setCode("");
            onOpenChange(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to submit code");
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-sm">
                <DialogHeader>
                    <DialogTitle>Telegram Authentication</DialogTitle>
                    <DialogDescription>
                        Enter the verification code sent to your Telegram account.
                    </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 pt-2">
                    <div className="space-y-2">
                        <Label htmlFor="auth-code">Verification Code</Label>
                        <input
                            id="auth-code"
                            type="text"
                            value={code}
                            onChange={(e) => setCode(e.target.value)}
                            placeholder="12345"
                            className="w-full rounded-md border px-3 py-2 text-sm"
                            autoFocus
                            onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
                        />
                    </div>
                    {error && (
                        <p className="text-sm text-destructive">{error}</p>
                    )}
                    <Button
                        className="w-full"
                        onClick={handleSubmit}
                        disabled={!code.trim() || submitting}
                    >
                        {submitting ? "Verifying..." : "Submit Code"}
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    );
}

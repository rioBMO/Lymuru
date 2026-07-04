import { useEffect, useState } from "react";
import { Copy, FolderOpen, Moon, Sun } from "lucide-react";
import {
  GetSettings,
  SaveSettings,
  PickFolder,
  OpenFolder,
  type Settings,
} from "@/lib/api";
import { useToast } from "@/components/Toast";
import { SidecarDiagnosticsCard } from "@/components/SidecarDiagnosticsCard";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface Props {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
}

const DEFAULTS: Settings = {
  theme_mode: "light",
  downloads_folder: "",
  has_completed_onboarding: false,
  python_path: "",
  export_lrc_file: true,
  ffmpeg_path: "",
  audio_source: "auto",
};

export function SettingsPage({ themeMode, onToggleTheme }: Props) {
  const { toast } = useToast();
  const [settings, setSettings] = useState<Settings>(DEFAULTS);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const s = (await GetSettings()) as unknown as Settings;
        setSettings({ ...DEFAULTS, ...s });
      } catch {
        /* keep defaults */
      }
      setLoading(false);
    })();
  }, []);

  async function handleChooseFolder() {
    try {
      const path = await PickFolder();
      if (!path) return;
      setSettings((s) => ({ ...s, downloads_folder: path }));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Failed to pick folder", "error");
    }
  }

  async function handleSave() {
    setSaving(true);
    try {
      await SaveSettings({
        ...settings,
        theme_mode: themeMode,
      });
      toast("Settings saved", "success");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Failed to save", "error");
    }
    setSaving(false);
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle>Appearance</CardTitle>
          <CardDescription>Switch between light and dark themes.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Button
              variant={themeMode === "light" ? "default" : "outline"}
              onClick={() => themeMode === "dark" && onToggleTheme()}
              className="gap-1.5"
            >
              <Sun className="h-4 w-4" />
              Light
            </Button>
            <Button
              variant={themeMode === "dark" ? "default" : "outline"}
              onClick={() => themeMode === "light" && onToggleTheme()}
              className="gap-1.5"
            >
              <Moon className="h-4 w-4" />
              Dark
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Downloads folder</CardTitle>
          <CardDescription>
            Where downloaded FLAC files are saved. The Telegram sidecar uses
            this folder when running.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="downloads-folder">Folder</Label>
            <div className="flex gap-2">
              <Input
                id="downloads-folder"
                readOnly
                value={settings.downloads_folder || ""}
                placeholder={loading ? "Loading…" : "Not set"}
                className="font-mono"
              />
              <Button
                variant="outline"
                onClick={handleChooseFolder}
                className="shrink-0"
              >
                <FolderOpen className="h-4 w-4" />
                Browse
              </Button>
              {settings.downloads_folder && (
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => OpenFolder(settings.downloads_folder)}
                  title="Open in file explorer"
                >
                  <Copy className="h-4 w-4" />
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Export .lrc file */}
      <Card>
        <CardHeader>
          <CardTitle>Lyrics export</CardTitle>
          <CardDescription>
            Save a separate <code className="bg-muted px-1 rounded">.lrc</code> (synced) or{" "}
            <code className="bg-muted px-1 rounded">.txt</code> (plain) file alongside downloaded audio.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={settings.export_lrc_file ?? true}
              onChange={(e) =>
                setSettings((s) => ({ ...s, export_lrc_file: e.target.checked }))
              }
              className="h-4 w-4 rounded border-border text-primary focus:ring-ring/30"
            />
            <span className="text-sm">Export .lrc file</span>
          </label>
        </CardContent>
      </Card>

      {/* FFmpeg executable path */}
      <Card>
        <CardHeader>
          <CardTitle>FFmpeg executable</CardTitle>
          <CardDescription>
            Path to the FFmpeg binary used for audio conversion. Leave empty
            to auto-detect from common install locations.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-1.5">
            <Label htmlFor="ffmpeg-path">FFmpeg executable (auto-detect if empty)</Label>
            <Input
              id="ffmpeg-path"
              value={settings.ffmpeg_path || ""}
              onChange={(e) => setSettings((s) => ({ ...s, ffmpeg_path: e.target.value }))}
              placeholder="ffmpeg (auto-detect)"
              className="font-mono"
            />
          </div>
        </CardContent>
      </Card>

      {/* Python executable path */}
      <Card>
        <CardHeader>
          <CardTitle>Python executable</CardTitle>
          <CardDescription>
            Path to the Python interpreter used by the sidecar. Leave empty
            to auto-detect from common install locations.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-1.5">
            <Label htmlFor="python-path">Executable path</Label>
            <Input
              id="python-path"
              value={settings.python_path || ""}
              onChange={(e) => setSettings((s) => ({ ...s, python_path: e.target.value }))}
              placeholder="python (auto-detect)"
              className="font-mono"
            />
          </div>
        </CardContent>
      </Card>

      {/* Sidecar diagnostics */}
      <SidecarDiagnosticsCard />

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={saving || loading}>
          {saving ? "Saving…" : "Save settings"}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>About</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Lymuru — a desktop app for downloading lossless music from Spotify &amp; Deezer via Telegram.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

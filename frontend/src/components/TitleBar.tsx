import { Maximize, Minus, X, SlidersHorizontal, Globe } from "lucide-react";
import { WindowMinimise, WindowToggleMaximise, Quit } from "../../wailsjs/runtime/runtime";
import { Menubar, MenubarContent, MenubarMenu, MenubarItem, MenubarTrigger, MenubarLabel, MenubarSeparator } from "@/components/ui/menubar";
import { Slider } from "@/components/ui/slider";
import { getSettings, updateSettings } from "@/lib/settings";
import { PREVIEW_VOLUME_CHANGED_EVENT } from "@/lib/preview";
import { openExternal } from "@/lib/utils";
import { useEffect, useState } from "react";
interface SettingsUpdatedDetail {
    previewVolume?: number;
}
export function TitleBar() {
    const initialSettings = getSettings();
    const [previewVolume, setPreviewVolume] = useState(initialSettings.previewVolume ?? 100);
    useEffect(() => {
        const handleSettingsUpdate = (event: Event) => {
            const updatedSettings = (event as CustomEvent<SettingsUpdatedDetail>).detail;
            if (updatedSettings && typeof updatedSettings.previewVolume === "number") {
                setPreviewVolume(updatedSettings.previewVolume);
            }
        };
        window.addEventListener("settingsUpdated", handleSettingsUpdate);
        return () => window.removeEventListener("settingsUpdated", handleSettingsUpdate);
    }, []);
    const handleMinimize = () => {
        WindowMinimise();
    };
    const handleMaximize = () => {
        WindowToggleMaximise();
    };
    const handleClose = () => {
        Quit();
    };
    const handlePreviewVolumeChange = (value: number[]) => {
        const nextValue = value[0];
        if (typeof nextValue !== "number" || Number.isNaN(nextValue)) {
            return;
        }
        setPreviewVolume(nextValue);
        window.dispatchEvent(new CustomEvent(PREVIEW_VOLUME_CHANGED_EVENT, { detail: nextValue }));
    };
    const handlePreviewVolumeCommit = (value: number[]) => {
        const nextValue = value[0];
        if (typeof nextValue !== "number" || Number.isNaN(nextValue)) {
            return;
        }
        setPreviewVolume(nextValue);
        void updateSettings({ previewVolume: nextValue });
    };
    return (<>

      <div className="fixed top-0 left-14 right-0 h-10 z-40 bg-background/80 backdrop-blur-sm" style={{ "--wails-draggable": "drag" } as React.CSSProperties} onDoubleClick={handleMaximize}/>


      <div className="fixed top-1.5 right-2 z-50 flex h-7 gap-0.5 items-center">
        <Menubar className="border-none bg-transparent shadow-none px-0 mr-1" style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}>
            <MenubarMenu>
                <MenubarTrigger className="cursor-pointer w-8 h-7 p-0 flex items-center justify-center hover:bg-muted transition-colors rounded data-[state=open]:bg-muted">
                    <SlidersHorizontal className="w-3.5 h-3.5"/>
                </MenubarTrigger>
                <MenubarContent align="end" className="min-w-70">
                    <div className="px-2 py-1.5 space-y-2">
                        <div className="flex items-center justify-between gap-3">
                            <MenubarLabel className="p-0">Preview Volume</MenubarLabel>
                            <span className="text-xs font-medium text-muted-foreground tabular-nums">
                                {previewVolume}%
                            </span>
                        </div>
                        <Slider value={[previewVolume]} min={0} max={100} step={5} onValueChange={handlePreviewVolumeChange} onValueCommit={handlePreviewVolumeCommit} aria-label="Preview volume"/>
                    </div>
                    <MenubarSeparator />
                    <MenubarItem onClick={() => openExternal("https://github.com/rioBMO/Lymuru")} className="gap-2">
                        <Globe className="w-4 h-4 opacity-70"/>
                        <span>Website</span>
                    </MenubarItem>
                </MenubarContent>
            </MenubarMenu>
        </Menubar>
        <button onClick={handleMinimize} className="w-8 h-7 flex items-center justify-center hover:bg-muted transition-colors rounded" style={{ "--wails-draggable": "no-drag" } as React.CSSProperties} aria-label="Minimize">
          <Minus className="w-3.5 h-3.5"/>
        </button>
        <button onClick={handleMaximize} className="w-8 h-7 flex items-center justify-center hover:bg-muted transition-colors rounded" style={{ "--wails-draggable": "no-drag" } as React.CSSProperties} aria-label="Maximize">
          <Maximize className="w-3.5 h-3.5"/>
        </button>
        <button onClick={handleClose} className="w-8 h-7 flex items-center justify-center hover:bg-destructive hover:text-white transition-colors rounded" style={{ "--wails-draggable": "no-drag" } as React.CSSProperties} aria-label="Close">
          <X className="w-3.5 h-3.5"/>
        </button>
      </div>
    </>);
}

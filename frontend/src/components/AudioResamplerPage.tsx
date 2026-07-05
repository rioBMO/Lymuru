import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Upload, X, CheckCircle2, AlertCircle, Trash2, FileMusic, WandSparkles } from "lucide-react";
import { Spinner } from "@/components/ui/spinner";
import { ResampleAudio, SelectAudioFiles, SelectFolder, ListAudioFilesInDir, GetFileSizes } from "../../wailsjs/go/main/App";
import { toastWithSound as toast } from "@/lib/toast-with-sound";

interface ResampleFile {
    path: string;
    name: string;
    format: string;
    size: number;
    status: "pending" | "resampling" | "success" | "error";
    error?: string;
    outputPath?: string;
}

function formatFileSize(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

const SAMPLE_RATES = [
    { value: "44100", label: "44.1 kHz" },
    { value: "48000", label: "48 kHz" },
    { value: "96000", label: "96 kHz" },
    { value: "192000", label: "192 kHz" },
];

const BIT_DEPTHS = [
    { value: "16", label: "16-bit" },
    { value: "24", label: "24-bit" },
];

export function AudioResamplerPage() {
    const [files, setFiles] = useState<ResampleFile[]>([]);
    const [sampleRate, setSampleRate] = useState<string>("44100");
    const [bitDepth, setBitDepth] = useState<string>("16");
    const [resampling, setResampling] = useState(false);

    const addFiles = useCallback(async (paths: string[]) => {
        const validExtensions = [".flac"];
        const validPaths = paths.filter((path) => {
            const ext = path.toLowerCase().slice(path.lastIndexOf("."));
            return validExtensions.includes(ext);
        });

        const fileSizes = validPaths.length > 0 ? await GetFileSizes(validPaths) : {};

        setFiles((prev) => {
            const newFiles: ResampleFile[] = validPaths
                .filter((path) => !prev.some((f) => f.path === path))
                .map((path) => {
                    const name = path.split(/[/\\]/).pop() || path;
                    return {
                        path,
                        name,
                        format: "flac",
                        size: fileSizes[path] || 0,
                        status: "pending" as const,
                    };
                });

            if (newFiles.length > 0) {
                if (paths.length > newFiles.length) {
                    const skipped = paths.length - newFiles.length;
                    toast.info("Some files skipped", {
                        description: `${skipped} file(s) were skipped (only FLAC files are supported)`,
                    });
                }
                return [...prev, ...newFiles];
            }
            toast.info("No new files added", {
                description: "All files were already added or have unsupported format",
            });
            return prev;
        });
    }, []);

    const handleSelectFiles = async () => {
        try {
            const selected = await SelectAudioFiles();
            if (selected && selected.length > 0) addFiles(selected);
        } catch (err) {
            toast.error("File Selection Failed", {
                description: err instanceof Error ? err.message : "Failed to select files",
            });
        }
    };

    const handleSelectFolder = async () => {
        try {
            const folder = await SelectFolder("");
            if (folder) {
                const folderFiles = await ListAudioFilesInDir(folder);
                if (folderFiles && folderFiles.length > 0) {
                    addFiles(folderFiles.map((f) => f.path));
                } else {
                    toast.info("No FLAC files found");
                }
            }
        } catch (err) {
            toast.error("Folder Selection Failed", {
                description: err instanceof Error ? err.message : "Failed to select folder",
            });
        }
    };

    const removeFile = (path: string) => {
        setFiles((prev) => prev.filter((f) => f.path !== path));
    };

    const clearFiles = () => {
        setFiles([]);
    };

    const handleResample = async () => {
        if (files.length === 0) {
            toast.error("No files selected");
            return;
        }

        setResampling(true);
        try {
            const inputPaths = files.map((f) => f.path);
            setFiles((prev) =>
                prev.map((f) => {
                    if (inputPaths.includes(f.path)) return { ...f, status: "resampling" as const, error: undefined };
                    return f;
                })
            );

            const results = await ResampleAudio({
                input_files: inputPaths,
                sample_rate: sampleRate,
                bit_depth: bitDepth,
            });

            setFiles((prev) =>
                prev.map((f) => {
                    const result = results.find(
                        (r) => r.input_file === f.path || r.input_file.toLowerCase() === f.path.toLowerCase()
                    );
                    if (result) {
                        return {
                            ...f,
                            status: result.success ? "success" : "error",
                            error: result.error,
                            outputPath: result.output_file,
                        } as ResampleFile;
                    }
                    return f;
                })
            );

            const ok = results.filter((r) => r.success).length;
            const fail = results.filter((r) => !r.success).length;
            if (ok > 0) {
                toast.success("Resampling Complete", {
                    description: `Resampled ${ok} file(s)${fail > 0 ? `, ${fail} failed` : ""}`,
                });
            } else if (fail > 0) {
                toast.error("Resampling Failed", { description: `All ${fail} file(s) failed` });
            }
        } catch (err) {
            toast.error("Resampling Error", {
                description: err instanceof Error ? err.message : "Unknown error",
            });
            setFiles((prev) =>
                prev.map((f) => ({ ...f, status: "error" as const, error: "Resampling failed" }))
            );
        } finally {
            setResampling(false);
        }
    };

    const getStatusIcon = (status: ResampleFile["status"]) => {
        switch (status) {
            case "resampling":
                return <Spinner className="h-4 w-4 text-primary" />;
            case "success":
                return <CheckCircle2 className="h-4 w-4 text-green-500" />;
            case "error":
                return <AlertCircle className="h-4 w-4 text-destructive" />;
            default:
                return <FileMusic className="h-4 w-4 text-muted-foreground" />;
        }
    };

    const canResample = files.filter((f) => f.status === "pending" || f.status === "success").length;
    const doneCount = files.filter((f) => f.status === "success").length;

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold">Audio Resampler</h1>
                {files.length > 0 && (
                    <div className="flex gap-2">
                        <Button variant="outline" size="sm" onClick={handleSelectFiles}><Upload className="h-4 w-4" />Add Files</Button>
                        <Button variant="outline" size="sm" onClick={handleSelectFolder}><Upload className="h-4 w-4" />Add Folder</Button>
                        <Button variant="outline" size="sm" onClick={clearFiles} disabled={resampling}><Trash2 className="h-4 w-4" />Clear All</Button>
                    </div>
                )}
            </div>

            <div className="flex flex-col items-center justify-center border-2 border-dashed rounded-lg transition-all h-[400px] border-muted-foreground/30">
                {files.length === 0 ? (
                    <>
                        <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                            <Upload className="h-8 w-8 text-primary" />
                        </div>
                        <p className="text-sm text-muted-foreground mb-4">Select FLAC files to resample</p>
                        <div className="flex gap-3">
                            <Button onClick={handleSelectFiles} size="lg"><Upload className="h-5 w-5" />Select Files</Button>
                            <Button onClick={handleSelectFolder} size="lg" variant="outline"><Upload className="h-5 w-5" />Select Folder</Button>
                        </div>
                        <p className="text-xs text-muted-foreground mt-4">Supports: FLAC files only</p>
                    </>
                ) : (
                    <div className="w-full h-full p-6 space-y-4 flex flex-col">
                        <div className="space-y-2 pb-4 border-b shrink-0">
                            <div className="flex items-center gap-4 flex-wrap">
                                <div className="flex items-center gap-2">
                                    <Label className="whitespace-nowrap">Sample Rate:</Label>
                                    <ToggleGroup type="single" variant="outline" value={sampleRate} onValueChange={(v) => v && setSampleRate(v)}>
                                        {SAMPLE_RATES.map((r) => <ToggleGroupItem key={r.value} value={r.value}>{r.label}</ToggleGroupItem>)}
                                    </ToggleGroup>
                                </div>
                                <div className="flex items-center gap-2">
                                    <Label className="whitespace-nowrap">Bit Depth:</Label>
                                    <ToggleGroup type="single" variant="outline" value={bitDepth} onValueChange={(v) => v && setBitDepth(v)}>
                                        {BIT_DEPTHS.map((r) => <ToggleGroupItem key={r.value} value={r.value}>{r.label}</ToggleGroupItem>)}
                                    </ToggleGroup>
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center justify-between shrink-0">
                            <div className="text-sm text-muted-foreground">{files.length} file(s) &bull; {doneCount} resampled</div>
                        </div>

                        <div className="flex-1 space-y-2 overflow-y-auto min-h-0">
                            {files.map((file) => (
                                <div key={file.path} className="flex items-center gap-3 rounded-lg border p-3">
                                    {getStatusIcon(file.status)}
                                    <div className="flex-1 min-w-0">
                                        <p className="truncate text-sm font-medium">{file.name}</p>
                                        {file.error && <p className="truncate text-xs text-destructive">{file.error}</p>}
                                    </div>
                                    <span className="text-xs text-muted-foreground">{formatFileSize(file.size)}</span>
                                    {file.status !== "resampling" && (
                                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => removeFile(file.path)} disabled={resampling}>
                                            <X className="h-4 w-4" />
                                        </Button>
                                    )}
                                </div>
                            ))}
                        </div>

                        <div className="flex justify-center pt-4 border-t shrink-0">
                            <Button onClick={handleResample} disabled={resampling || canResample === 0} size="lg">
                                {resampling ? (<><Spinner className="h-4 w-4" /> Resampling...</>) : (<><WandSparkles className="h-4 w-4" /> Resample {canResample > 0 ? `${canResample} File(s)` : ""}</>)}
                            </Button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}

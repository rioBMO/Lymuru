import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Upload, FolderOpen, FileMusic, X } from "lucide-react";
import { Spinner } from "@/components/ui/spinner";
import { AudioAnalysis } from "@/components/AudioAnalysis";
import { AnalyzeAudio, SelectAudioFiles, SelectFolder, ListAudioFilesInDir, GetFileSizes } from "../../wailsjs/go/main/App";
import { toastWithSound as toast } from "@/lib/toast-with-sound";
import type { backend } from "../../wailsjs/go/models";

interface AnalysisFile {
    path: string;
    name: string;
    size: number;
    status: "pending" | "analyzing" | "done" | "error";
    result?: backend.AnalysisResult;
    error?: string;
}

function formatFileSize(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

export function AudioAnalysisPage() {
    const [files, setFiles] = useState<AnalysisFile[]>([]);
    const [selectedFile, setSelectedFile] = useState<AnalysisFile | null>(null);
    const [analyzing, setAnalyzing] = useState(false);

    const addFiles = async (paths: string[]) => {
        const validExts = [".flac", ".mp3", ".m4a", ".aac"];
        const validPaths = paths.filter((p) => {
            const ext = p.toLowerCase().slice(p.lastIndexOf("."));
            return validExts.includes(ext);
        });

        const fileSizes = validPaths.length > 0 ? await GetFileSizes(validPaths) : {};

        setFiles((prev) => {
            const newFiles: AnalysisFile[] = validPaths
                .filter((p) => !prev.some((f) => f.path === p))
                .map((p) => ({
                    path: p,
                    name: p.split(/[/\\]/).pop() || p,
                    size: fileSizes[p] || 0,
                    status: "pending" as const,
                }));
            return [...prev, ...newFiles];
        });
    };

    const handleSelectFiles = async () => {
        try {
            const selected = await SelectAudioFiles();
            if (selected && selected.length > 0) {
                addFiles(selected);
            }
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
                    toast.info("No audio files found", {
                        description: "No supported audio files found in the selected folder.",
                    });
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
        if (selectedFile?.path === path) {
            setSelectedFile(null);
        }
    };

    const clearAll = () => {
        setFiles([]);
        setSelectedFile(null);
    };

    const handleAnalyze = async (file: AnalysisFile) => {
        setSelectedFile(file);
        setAnalyzing(true);
        setFiles((prev) =>
            prev.map((f) => (f.path === file.path ? { ...f, status: "analyzing" as const, error: undefined } : f))
        );
        try {
            const result = await AnalyzeAudio(file.path);
            const updated: AnalysisFile = { ...file, status: "done" as const, result };
            setFiles((prev) => prev.map((f) => (f.path === file.path ? updated : f)));
            setSelectedFile(updated);
        } catch (err) {
            const msg = err instanceof Error ? err.message : "Analysis failed";
            const updated: AnalysisFile = { ...file, status: "error" as const, error: msg };
            setFiles((prev) => prev.map((f) => (f.path === file.path ? updated : f)));
            setSelectedFile(updated);
            toast.error("Analysis Failed", { description: msg });
        } finally {
            setAnalyzing(false);
        }
    };

    const handleReanalyze = () => {
        if (selectedFile) {
            handleAnalyze(selectedFile);
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-bold">Audio Analysis</h1>
                <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={handleSelectFiles}>
                        <Upload className="h-4 w-4" />
                        Add Files
                    </Button>
                    <Button variant="outline" size="sm" onClick={handleSelectFolder}>
                        <FolderOpen className="h-4 w-4" />
                        Add Folder
                    </Button>
                    {files.length > 0 && (
                        <Button variant="outline" size="sm" onClick={clearAll}>
                            <X className="h-4 w-4" />
                            Clear All
                        </Button>
                    )}
                </div>
            </div>

            {files.length === 0 ? (
                <div className="flex flex-col items-center justify-center border-2 border-dashed rounded-lg h-[400px] border-muted-foreground/30">
                    <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
                        <FileMusic className="h-8 w-8 text-primary" />
                    </div>
                    <p className="text-sm text-muted-foreground mb-4 text-center">
                        Select audio files to analyze their quality
                    </p>
                    <div className="flex gap-3">
                        <Button onClick={handleSelectFiles} size="lg">
                            <Upload className="h-5 w-5" />
                            Select Files
                        </Button>
                        <Button onClick={handleSelectFolder} size="lg" variant="outline">
                            <FolderOpen className="h-5 w-5" />
                            Select Folder
                        </Button>
                    </div>
                    <p className="text-xs text-muted-foreground mt-4 text-center">
                        Supported formats: FLAC, MP3, M4A, AAC
                    </p>
                </div>
            ) : (
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* File list sidebar */}
                    <div className="lg:col-span-1 border rounded-lg">
                        <div className="p-3 border-b flex items-center justify-between">
                            <span className="text-sm font-medium">{files.length} file(s)</span>
                        </div>
                        <div className="max-h-[500px] overflow-y-auto">
                            {files.map((file) => (
                                <button
                                    key={file.path}
                                    className={`w-full text-left p-3 border-b last:border-b-0 flex items-center gap-2 hover:bg-muted/50 transition-colors ${selectedFile?.path === file.path ? "bg-primary/10 border-l-2 border-l-primary" : ""}`}
                                    onClick={() => {
                                        setSelectedFile(file);
                                        if (file.status === "pending") {
                                            handleAnalyze(file);
                                        }
                                    }}
                                >
                                    <FileMusic className="h-4 w-4 text-muted-foreground shrink-0" />
                                    <div className="min-w-0 flex-1">
                                        <p className="truncate text-sm">{file.name}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {formatFileSize(file.size)}
                                            {file.status === "analyzing" && " \u2022 Analyzing..."}
                                            {file.status === "error" && " \u2022 Failed"}
                                        </p>
                                    </div>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-6 w-6 shrink-0"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            removeFile(file.path);
                                        }}
                                    >
                                        <X className="h-3 w-3" />
                                    </Button>
                                </button>
                            ))}
                        </div>
                    </div>

                    {/* Analysis results panel */}
                    <div className="lg:col-span-2">
                        {!selectedFile && (
                            <div className="flex flex-col items-center justify-center border rounded-lg h-[400px] text-muted-foreground">
                                <FileMusic className="h-12 w-12 mb-3 opacity-30" />
                                <p>Select a file from the list to analyze</p>
                            </div>
                        )}
                        {selectedFile && (
                            <AudioAnalysis
                                result={selectedFile.result ?? null}
                                analyzing={analyzing && selectedFile.status === "analyzing"}
                                onAnalyze={selectedFile.status !== "analyzing" ? handleReanalyze : undefined}
                                showAnalyzeButton={selectedFile.status === "done"}
                                filePath={selectedFile.path}
                            />
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

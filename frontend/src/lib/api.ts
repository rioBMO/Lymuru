/**
 * Wails binding wrapper.
 *
 * This file replaces the previous `fetch()`-based HTTP API wrapper.
 * All backend calls go through Wails-generated TypeScript bindings
 * that are auto-generated into `frontend/wailsjs/go/main/App` by
 * `wails dev` / `wails build`.
 *
 * Until the Wails CLI is run and bindings are generated, the
 * `wailsjs` modules will not exist. To run the frontend standalone
 * (e.g. via `bun run dev` without Wails), stub the imports below
 * or run `wails generate module` to materialize the bindings.
 */

import {
  Search,
  Download,
  DownloadLink,
  ChooseLyrics,
  GetHistory,
  DeleteHistoryItem,
  ClearHistory,
  GetActiveTasks,
  CancelTask,
  GetSettings,
  SaveSettings,
  GetVersion,

  PickFile,
  PickFolder,
  OpenFolder,
  GetDownloadsPath,
  AddLyrics,
  EmbedLrc,
  RomanizeLrc,
  ExtractLrc,
} from "../../wailsjs/go/main/App";

// Re-export so components can import from `@/lib/api`.
export {
  Search,
  Download,
  DownloadLink,
  ChooseLyrics,
  GetHistory,
  DeleteHistoryItem,
  ClearHistory,
  GetActiveTasks,
  CancelTask,
  GetSettings,
  SaveSettings,
  GetVersion,

  PickFile,
  PickFolder,
  OpenFolder,
  GetDownloadsPath,
  AddLyrics,
  EmbedLrc,
  RomanizeLrc,
  ExtractLrc,
};

// ---------------------------------------------------------------------------
// Type aliases matching the Go-side bindings.
// (Kept in sync with the App struct in app.go.)
// ---------------------------------------------------------------------------

export interface SearchResult {
  index: number;
  title: string;
  description: string;
}

export interface SearchResponse {
  results: SearchResult[];
  search_key: string;
}

export interface DownloadResponse {
  task_id: string;
}

export interface HistoryEntry {
  id: number;
  task_id: string;
  task_type: string;
  query: string;
  status: "completed" | "failed";
  files: string[];
  error?: string;
  created_at: string;
}

export interface HistoryResponse {
  entries: HistoryEntry[];
  total: number;
}

export interface ActiveTask {
  task_id: string;
  task_type: string;
  query: string;
  stage: string;
  phase: string;
  download_percent: number;
  files?: string[];
  error?: string;
  created_at: string;
}

export interface Settings {
  theme_mode: "light" | "dark";
  downloads_folder: string;
  has_completed_onboarding: boolean;
  export_lrc_file: boolean;
  ffmpeg_path: string;
  audio_source: string;
}


export interface RomanizeResult {
  romanized: string | null;
  download_url: string | null;
  message: string;
}

export interface ExtractResult {
  lyrics: string;
  is_synced: boolean;
  output_url: string;
}

// ---------------------------------------------------------------------------
// Wails event names — keep in sync with backend/progress.go.
// ---------------------------------------------------------------------------

export const Events = {
  TaskProgress: "task:progress",
  TaskComplete: "task:complete",
  TaskError:    "task:error",
} as const;

export interface TaskProgressPayload {
  task_id: string;
  stage: string;
  phase: string;
  download_percent: number;
  download_received: number;
  download_total: number;
  query: string;
  task_type: string;
}

export interface TaskCompletePayload {
  task_id: string;
  files: string[];
}

export interface TaskErrorPayload {
  task_id: string;
  message: string;
}



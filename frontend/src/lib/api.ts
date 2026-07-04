/**
 * Wails binding wrapper.
 */

import {
  GetHistory,
  DeleteHistoryItem,
  ClearHistory,
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
  
  GetSpotifyMetadata, 
  DownloadTrack,
  SearchSpotify,
  SearchSpotifyByType,
  GetRecentFetches,
  SaveRecentFetches
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";

export async function fetchSpotifyMetadata(url: string, batch: boolean = true, delay: number = 1.0, timeout: number = 300.0) {
    const req = new main.SpotifyMetadataRequest({
        url,
        batch,
        delay,
        timeout,
    });
    const jsonString = await GetSpotifyMetadata(req);
    return JSON.parse(jsonString);
}

export async function downloadTrack(request: any) {
    const req = new main.DownloadRequest(request);
    if (request.use_single_genre !== undefined) {
        (req as any).use_single_genre = request.use_single_genre;
    }
    return await DownloadTrack(req);
}

export {
  GetHistory,
  DeleteHistoryItem,
  ClearHistory,
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
  
  SearchSpotify,
  SearchSpotifyByType,
  GetRecentFetches,
  SaveRecentFetches
};

// Types for History
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
export interface Settings {
  theme_mode: "light" | "dark";
  downloads_folder: string;
  has_completed_onboarding: boolean;
  export_lrc_file: boolean;
  ffmpeg_path: string;
  audio_source: string;
}

export const Events = {
  TaskProgress: "task:progress",
  TaskComplete: "task:complete",
  TaskError:    "task:error",
} as const;

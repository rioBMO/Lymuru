package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lymuru/lymuru/backend"
	"github.com/lymuru/lymuru/backend/storage"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application object. Methods on App are exposed to
// the React frontend as JavaScript functions.
type App struct {
	ctx     context.Context
	mu      sync.Mutex
	storage *storage.DB
	tasks   *backend.TaskManager
	history *backend.History
	config  *backend.Config
	sidecar *backend.DeezerSidecar
	started bool
}

// NewApp creates a new App. Call SetStorage before startup.
func NewApp() *App {
	return &App{
		tasks:   backend.NewTaskManager(),
		started: false,
	}
}

// SetStorage wires the SQLite database into the App. Must be called before
// the Wails startup hook.
func (a *App) SetStorage(db *storage.DB) {
	a.storage = db
	a.history = backend.NewHistory(db)
	a.config = backend.NewConfig(db)
}

// startup is called by Wails after the WebView is ready. It boots the
// sidecar subprocess and seeds in-memory state.
func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.started = true
	a.mu.Unlock()

	// Start the Deezer sidecar if enabled.
	go func() {
		settings, err := a.config.Load()
		if err != nil {
			backend.LogWarn("[Sidecar] config load failed: %v", err)
			return
		}
		if !settings.DeezerEnabled {
			return
		}
		a.sidecar = backend.NewDeezerSidecar("data", settings.PythonPath)
		if err := a.sidecar.Start(); err != nil {
			backend.LogWarn("[Sidecar] start failed: %v", err)
			wailsruntime.EventsEmit(a.ctx, "sidecar:status", backend.SidecarStatus{
				Running: false,
				Error:   err.Error(),
			})
			return
		}
		// Forward sidecar events to the frontend.
		go func() {
			for ev := range a.sidecar.EventChan() {
				wailsruntime.EventsEmit(a.ctx, "sidecar:event", ev)
			}
		}()
		wailsruntime.EventsEmit(a.ctx, "sidecar:status", a.sidecar.Status())
	}()
}

// shutdown is called by Wails before the window closes.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	a.started = false
	a.mu.Unlock()
	if a.sidecar != nil {
		a.sidecar.Stop()
	}
	if a.storage != nil {
		_ = a.storage.Close()
	}
}

// ---------------------------------------------------------------------------
// Bound methods (Wails generates JS bindings from these)
// ---------------------------------------------------------------------------

// SearchResult mirrors a single search hit from the sidecar.
type SearchResult struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// SearchResponse is the payload returned by Search.
type SearchResponse struct {
	Results   []SearchResult `json:"results"`
	SearchKey string         `json:"search_key"`
}

// SidecarTaskResponse is returned by sidecar-queued tasks.
type SidecarTaskResponse struct {
	TaskID string `json:"task_id"`
}

// DownloadFFmpegResponse is returned by the DownloadFFmpeg binding.
type DownloadFFmpegResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ConvertAudioRequest mirrors backend.ConvertAudioRequest for Wails binding.
type ConvertAudioRequest struct {
	InputFiles   []string `json:"input_files"`
	OutputFormat string   `json:"output_format"`
	Bitrate      string   `json:"bitrate"`
	Codec        string   `json:"codec"`
}

// ------------------------------------------
// FFmpeg
// ------------------------------------------

func (a *App) CheckFFmpegInstalled() (bool, error) {
	return backend.IsFFmpegInstalled()
}

func (a *App) DownloadFFmpeg() DownloadFFmpegResponse {
	wailsruntime.EventsEmit(a.ctx, "ffmpeg:status", "starting")
	err := backend.DownloadFFmpeg(func(progress int) {
		wailsruntime.EventsEmit(a.ctx, "ffmpeg:progress", progress)
	})
	if err != nil {
		wailsruntime.EventsEmit(a.ctx, "ffmpeg:status", "failed")
		return DownloadFFmpegResponse{
			Success: false,
			Error:   err.Error(),
		}
	}
	wailsruntime.EventsEmit(a.ctx, "ffmpeg:status", "completed")
	return DownloadFFmpegResponse{
		Success: true,
	}
}

// History stubs — return empty results until real SQLite history is wired.
func (a *App) GetDownloadHistory() []backend.DownloadHistoryItem {
	items, err := a.history.GetDownloadHistoryItems()
	if err != nil {
		return nil
	}
	return items
}
func (a *App) ClearDownloadHistory() error { return a.history.ClearDownloadHistory() }
func (a *App) GetFetchHistory() []backend.FetchHistoryItem {
	items, err := a.history.GetFetchHistoryItems()
	if err != nil {
		return nil
	}
	return items
}
func (a *App) DeleteDownloadHistoryItem(id string) error {
	return a.history.DeleteDownloadHistoryItem(id)
}
func (a *App) DeleteFetchHistoryItem(id string) error { return a.history.DeleteFetchHistoryItem(id) }
func (a *App) ClearFetchHistoryByType(type_ string) error {
	return a.history.ClearFetchHistoryByType(type_)
}

// Lyrics Manager

func (a *App) ReadEmbeddedLyrics(filePath string) (*backend.EmbeddedLyrics, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	return backend.ReadEmbeddedLyrics(filePath)
}

func (a *App) ExtractLyricsToLRC(filePath string, overwrite bool) (*backend.ExtractLyricsResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	return backend.ExtractLyricsToLRC(filePath, overwrite)
}

func (a *App) SelectLyricsFiles() ([]string, error) {
	files, err := backend.SelectLyricsFiles(a.ctx)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (a *App) SelectLyricsFolder() (string, error) {
	return backend.SelectLyricsFolder(a.ctx)
}

func (a *App) ScanLyricsFolder(dir string) ([]string, error) {
	if dir == "" {
		return nil, fmt.Errorf("folder path is required")
	}
	return backend.ScanLyricsFolder(dir)
}

func (a *App) SaveLyrics(filePath string, lyrics string) (*backend.SaveLyricsResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	return backend.SaveLyrics(filePath, lyrics)
}

// GetDefaults returns default settings values. The frontend uses the
// downloadPath field when no path has been configured yet.
func (a *App) GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"downloadPath": backend.DefaultSettings().DownloadsFolder,
	}
}

// LoadSettings returns settings from persistent storage. Returns nil on error
// so the frontend falls back to localStorage.
func (a *App) LoadSettings() map[string]interface{} {
	s, err := a.config.Load()
	if err != nil {
		return nil
	}
	return map[string]interface{}{
		"theme_mode":               s.ThemeMode,
		"downloads_folder":         s.DownloadsFolder,
		"has_completed_onboarding": s.HasCompletedOnboarding,
		"export_lrc_file":          s.ExportLrcFile,
		"ffmpeg_path":              s.FFmpegPath,
		"audio_source":             s.AudioSource,
		"audio_format":             s.AudioFormat,
		"filename_format":          s.FilenameFormat,
		"customTidalApi":           s.CustomTidalAPI,
		"customQobuzApi":           s.CustomQobuzAPI,
		"existing_file_check_mode": s.ExistingFileCheckMode,
		"link_resolver":            s.LinkResolver,
		"auto_order":               s.AutoOrder,
		"separator":                s.Separator,
		"deezer_enabled":           s.DeezerEnabled,
		"python_path":              s.PythonPath,
	}
}

// LoadFonts returns custom font definitions from persistent storage.
func (a *App) LoadFonts() []map[string]interface{} {
	return a.config.LoadFonts()
}

// SaveFonts persists custom font definitions.
func (a *App) SaveFonts(f []map[string]interface{}) error {
	return a.config.SaveFonts(f)
}

// ------------------------------------------

// GetVersion returns the app version.
func (a *App) GetVersion() string {
	return backend.AppVersion()
}

// GetTask returns a snapshot of a task by id.
func (a *App) GetTask(taskID string) (backend.Task, error) {
	t, ok := a.tasks.Get(taskID)
	if !ok {
		return backend.Task{}, fmt.Errorf("task not found: %s", taskID)
	}
	return t, nil
}

// GetActiveTasks returns the list of active tasks for the queue dialog.
func (a *App) GetActiveTasks() []backend.Task {
	return a.tasks.List()
}

// CancelTask marks a task as cancelled. The sidecar is expected to honor
// this by stopping the underlying work.
func (a *App) CancelTask(taskID string) error {
	return a.tasks.Cancel(taskID)
}

// GetHistory returns paginated history rows.
func (a *App) GetHistory(limit, offset int, status, search string) (backend.HistoryResponse, error) {
	return a.history.List(limit, offset, status, search)
}

// DeleteHistoryItem deletes a single history entry.
func (a *App) DeleteHistoryItem(id int64) error {
	return a.history.Delete(id)
}

// ClearHistory removes all history.
func (a *App) ClearHistory() error {
	return a.history.Clear()
}

// GetSettings returns the current settings as a map.
func (a *App) GetSettings() map[string]interface{} {
	s, err := a.config.Load()
	if err != nil {
		defaults := backend.DefaultSettings()
		return map[string]interface{}{
			"theme_mode":               defaults.ThemeMode,
			"downloads_folder":         defaults.DownloadsFolder,
			"has_completed_onboarding": defaults.HasCompletedOnboarding,
			"export_lrc_file":          defaults.ExportLrcFile,
			"ffmpeg_path":              defaults.FFmpegPath,
			"audio_source":             defaults.AudioSource,
			"audio_format":             defaults.AudioFormat,
			"filename_format":          defaults.FilenameFormat,
			"customTidalApi":           defaults.CustomTidalAPI,
			"customQobuzApi":           defaults.CustomQobuzAPI,
			"existing_file_check_mode": defaults.ExistingFileCheckMode,
			"link_resolver":            defaults.LinkResolver,
			"auto_order":               defaults.AutoOrder,
			"separator":                defaults.Separator,
			"deezerEnabled":            defaults.DeezerEnabled,
			"pythonPath":               defaults.PythonPath,
		}
	}
	return map[string]interface{}{
		"theme_mode":               s.ThemeMode,
		"downloads_folder":         s.DownloadsFolder,
		"has_completed_onboarding": s.HasCompletedOnboarding,
		"export_lrc_file":          s.ExportLrcFile,
		"ffmpeg_path":              s.FFmpegPath,
		"audio_source":             s.AudioSource,
		"audio_format":             s.AudioFormat,
		"filename_format":          s.FilenameFormat,
		"customTidalApi":           s.CustomTidalAPI,
		"customQobuzApi":           s.CustomQobuzAPI,
		"existing_file_check_mode": s.ExistingFileCheckMode,
		"link_resolver":            s.LinkResolver,
		"auto_order":               s.AutoOrder,
		"separator":                s.Separator,
	}
}

// SaveSettings persists settings from a map (SpoitFLAC-compatible).
// Accepts both snake_case and camelCase keys for robustness.
func (a *App) SaveSettings(s map[string]interface{}) error {
	var bs backend.Settings
	// --- required fields ---
	if v, ok := s["downloads_folder"].(string); ok && v != "" {
		bs.DownloadsFolder = v
	} else if v, ok := s["downloadPath"].(string); ok && v != "" {
		bs.DownloadsFolder = v
	}
	if v, ok := s["theme_mode"].(string); ok {
		bs.ThemeMode = v
	} else if v, ok := s["themeMode"].(string); ok {
		bs.ThemeMode = v
	}
	if v, ok := s["has_completed_onboarding"].(bool); ok {
		bs.HasCompletedOnboarding = v
	} else if v, ok := s["hasCompletedOnboarding"].(bool); ok {
		bs.HasCompletedOnboarding = v
	}
	if v, ok := s["export_lrc_file"].(bool); ok {
		bs.ExportLrcFile = v
	} else if v, ok := s["exportLrcFile"].(bool); ok {
		bs.ExportLrcFile = v
	}
	if v, ok := s["ffmpeg_path"].(string); ok {
		bs.FFmpegPath = v
	} else if v, ok := s["ffmpegPath"].(string); ok {
		bs.FFmpegPath = v
	}
	if v, ok := s["audio_source"].(string); ok {
		bs.AudioSource = v
	} else if v, ok := s["audioSource"].(string); ok {
		bs.AudioSource = v
	} else if v, ok := s["downloader"].(string); ok {
		bs.AudioSource = v
	}
	// --- additional backend fields ---
	if v, ok := s["audio_format"].(string); ok {
		bs.AudioFormat = v
	} else if v, ok := s["audioFormat"].(string); ok {
		bs.AudioFormat = v
	}
	if v, ok := s["filename_format"].(string); ok {
		bs.FilenameFormat = v
	} else if v, ok := s["filenameFormat"].(string); ok {
		bs.FilenameFormat = v
	}
	if v, ok := s["customTidalApi"].(string); ok {
		bs.CustomTidalAPI = v
	} else if v, ok := s["custom_tidal_api"].(string); ok {
		bs.CustomTidalAPI = v
	}
	if v, ok := s["customQobuzApi"].(string); ok {
		bs.CustomQobuzAPI = v
	} else if v, ok := s["custom_qobuz_api"].(string); ok {
		bs.CustomQobuzAPI = v
	}
	if v, ok := s["existing_file_check_mode"].(string); ok {
		bs.ExistingFileCheckMode = v
	} else if v, ok := s["existingFileCheckMode"].(string); ok {
		bs.ExistingFileCheckMode = v
	}
	if v, ok := s["link_resolver"].(string); ok {
		bs.LinkResolver = v
	} else if v, ok := s["linkResolver"].(string); ok {
		bs.LinkResolver = v
	}
	if v, ok := s["auto_order"].(string); ok {
		bs.AutoOrder = v
	} else if v, ok := s["autoOrder"].(string); ok {
		bs.AutoOrder = v
	}
	if v, ok := s["separator"].(string); ok {
		bs.Separator = v
	}
	// Sidecar / Deezer settings.
	if v, ok := s["deezer_enabled"].(bool); ok {
		bs.DeezerEnabled = v
	}
	if v, ok := s["python_path"].(string); ok {
		bs.PythonPath = v
	} else if v, ok := s["pythonPath"].(string); ok {
		bs.PythonPath = v
	}
	if bs.DownloadsFolder != "" {
		if err := backend.EnsureDownloadsFolder(bs.DownloadsFolder); err != nil {
			return err
		}
	}
	return a.config.Save(bs)
}

// AddLyrics searches and embeds lyrics directly.
func (a *App) AddLyrics(filePath, artist, title string) (string, error) {
	lyrics, synced, err := backend.SearchLRCLIB(artist, title)
	if err != nil {
		return "", err
	}
	err = backend.EmbedLyrics(filePath, lyrics)
	if err != nil {
		return "", err
	}
	syncStr := "unsynced"
	if synced {
		syncStr = "synced"
	}
	return fmt.Sprintf("Embedded %s lyrics", syncStr), nil
}

// EmbedLrc embeds a local LRC file directly.
func (a *App) EmbedLrc(flacPath, lrcPath string) (string, error) {
	lrcBytes, err := os.ReadFile(lrcPath)
	if err != nil {
		return "", err
	}
	err = backend.EmbedLyrics(flacPath, string(lrcBytes))
	if err != nil {
		return "", err
	}
	return "LRC embedded successfully", nil
}

// RomanizeResult is the payload returned by RomanizeLrc.
type RomanizeResult struct {
	Romanized   string `json:"romanized"`
	DownloadURL string `json:"download_url"`
	Message     string `json:"message"`
}

// RomanizeLrc romanizes an LRC file.
func (a *App) RomanizeLrc(lrcPath string) (RomanizeResult, error) {
	lrcBytes, err := os.ReadFile(lrcPath)
	if err != nil {
		return RomanizeResult{}, err
	}
	romanized, changed := backend.RomanizeLyrics(string(lrcBytes))
	if !changed {
		return RomanizeResult{Message: "No CJK lyrics found for romanization"}, nil
	}

	outPath := strings.TrimSuffix(lrcPath, ".lrc") + "_rom.lrc"
	if err := os.WriteFile(outPath, []byte(romanized), 0644); err != nil {
		return RomanizeResult{}, err
	}
	return RomanizeResult{
		Romanized:   romanized,
		DownloadURL: outPath,
		Message:     "Romanized successfully",
	}, nil
}

// ExtractResult is the payload returned by ExtractLrc.
type ExtractResult struct {
	Lyrics    string `json:"lyrics"`
	IsSynced  bool   `json:"is_synced"`
	OutputURL string `json:"output_url"`
}

// ExtractLrc reads lyrics from the file's tags and writes to an LRC file.
func (a *App) ExtractLrc(flacPath string) (ExtractResult, error) {
	lyrics, err := backend.ExtractLyrics(flacPath)
	if err != nil {
		return ExtractResult{}, err
	}
	outPath := strings.TrimSuffix(flacPath, filepath.Ext(flacPath)) + ".lrc"
	if err := os.WriteFile(outPath, []byte(lyrics), 0644); err != nil {
		return ExtractResult{}, err
	}
	return ExtractResult{
		Lyrics:    lyrics,
		IsSynced:  strings.Contains(lyrics, "[00:"),
		OutputURL: outPath,
	}, nil
}

// GetDownloadsPath returns the configured downloads folder.
func (a *App) GetDownloadsPath() string {
	s, err := a.config.Load()
	if err != nil {
		return backend.DefaultSettings().DownloadsFolder
	}
	return s.DownloadsFolder
}

// OpenFolder opens a path in the OS file explorer.
func (a *App) OpenFolder(path string) error {
	if path == "" {
		return errors.New("path is required")
	}
	wd, _ := os.Getwd()
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		// Fall back to the default downloads folder.
		defaults := backend.DefaultSettings()
		abs = defaults.DownloadsFolder
		_ = os.MkdirAll(abs, 0o755)
	}
	if a.ctx != nil {
		wailsruntime.BrowserOpenURL(a.ctx, "file://"+filepath.ToSlash(abs))
		return nil
	}
	_ = wd
	return nil
}

// PickFile opens a native file dialog and returns the selected path.
func (a *App) PickFile(filterDescription, filterPattern string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not ready")
	}
	filters := []wailsruntime.FileFilter{
		{DisplayName: filterDescription, Pattern: filterPattern},
	}
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:                "Select file",
		Filters:              filters,
		ShowHiddenFiles:      false,
		CanCreateDirectories: false,
	})
}

// PickFolder opens a native folder dialog and returns the selected path.
func (a *App) PickFolder() (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not ready")
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:                "Select folder",
		ShowHiddenFiles:      false,
		CanCreateDirectories: false,
	})
}

// SelectFolder opens a native folder dialog with a default path (SpoitFLAC settings page).
func (a *App) SelectFolder(defaultPath string) (string, error) {
	return backend.SelectFolderDialog(a.ctx, defaultPath)
}

// OpenConfigFolder opens the Lymuru data/config directory in the OS file explorer.
func (a *App) OpenConfigFolder() error {
	// Open the data directory where lymuru.db and config live.
	cwd, _ := os.Getwd()
	configDir := filepath.Join(cwd, "data")
	if _, err := os.Stat(configDir); err != nil {
		_ = os.MkdirAll(configDir, 0o755)
	}
	return a.OpenFolder(configDir)
}

// CheckCustomTidalAPI verifies a custom Tidal community API endpoint is reachable.
func (a *App) CheckCustomTidalAPI(apiURL string) (bool, error) {
	return backend.SimpleHealthCheck(apiURL)
}

// CheckCustomQobuzAPI verifies a custom Qobuz community API endpoint is reachable.
func (a *App) CheckCustomQobuzAPI(apiURL string) (bool, error) {
	return backend.SimpleHealthCheck(apiURL)
}

// ClearCommunityCooldown clears the global community API cooldown state.
// Called by the frontend when a download succeeds during auto-fallback
// to dismiss the cooldown banner.
func (a *App) ClearCommunityCooldown() {
	backend.ClearCommunityCooldown()
}

// ---------------------------------------------------------------------------
// File Manager bindings
// ---------------------------------------------------------------------------

// ListDirectoryFiles returns the recursive contents of a directory.
func (a *App) ListDirectoryFiles(dirPath string) ([]backend.FileInfo, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path is required")
	}
	return backend.ListDirectory(dirPath)
}

// ReadFileMetadata reads audio metadata (title, artist, album, etc.) from a file.
func (a *App) ReadFileMetadata(filePath string) (*backend.AudioMetadata, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	return backend.ReadAudioMetadata(filePath)
}

// ReadTextFile reads a text file and returns its contents.
func (a *App) ReadTextFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ReadImageAsBase64 reads an image file and returns a base64 data URI.
func (a *App) ReadImageAsBase64(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var mimeType string
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	default:
		mimeType = "image/jpeg"
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// PreviewRenameFiles generates filename previews for a batch rename operation.
func (a *App) PreviewRenameFiles(files []string, format string) []backend.RenamePreview {
	return backend.PreviewRename(files, format)
}

// RenameFilesByMetadata executes a batch rename using audio metadata.
func (a *App) RenameFilesByMetadata(files []string, format string) []backend.RenameResult {
	return backend.RenameFiles(files, format)
}

// RenameFileTo renames a single file to the given new name (without extension).
func (a *App) RenameFileTo(oldPath, newName string) error {
	dir := filepath.Dir(oldPath)
	ext := filepath.Ext(oldPath)
	newPath := filepath.Join(dir, newName+ext)
	return os.Rename(oldPath, newPath)
}

// ---------------------------------------------------------------------------
// Audio tools bindings
// ---------------------------------------------------------------------------

// SelectAudioFiles opens a file picker for selecting multiple audio files.
func (a *App) SelectAudioFiles() ([]string, error) {
	return backend.SelectMultipleFiles(a.ctx)
}

// ListAudioFilesInDir scans a directory for audio files and returns their info.
func (a *App) ListAudioFilesInDir(dirPath string) ([]backend.FileInfo, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path is required")
	}
	return backend.ListAudioFiles(dirPath)
}

// GetFileSizes returns the file size in bytes for each path.
func (a *App) GetFileSizes(files []string) map[string]int64 {
	return backend.GetFileSizes(files)
}

// ConvertAudio converts one or more audio files to the requested format.
func (a *App) ConvertAudio(req ConvertAudioRequest) ([]backend.ConvertAudioResult, error) {
	if len(req.InputFiles) == 0 {
		return nil, fmt.Errorf("no input files provided")
	}
	backendReq := backend.ConvertAudioRequest{
		InputFiles:   req.InputFiles,
		OutputFormat: req.OutputFormat,
		Bitrate:      req.Bitrate,
		Codec:        req.Codec,
	}
	return backend.ConvertAudio(backendReq)
}

// AnalyzeAudio returns ffprobe-derived audio quality metrics for a file.
func (a *App) AnalyzeAudio(filePath string) (*backend.AnalysisResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	return backend.GetMetadataWithFFprobe(filePath)
}

// ---------------------------------------------------------------------------
// Sidecar / Deezer bindings
// ---------------------------------------------------------------------------

// GetSidecarStatus returns the current sidecar state.
func (a *App) GetSidecarStatus() backend.SidecarStatus {
	if a.sidecar == nil {
		return backend.SidecarStatus{Running: false}
	}
	return a.sidecar.Status()
}

// SubmitSidecarAuthCode forwards a Telegram auth code to the sidecar.
func (a *App) SubmitSidecarAuthCode(code string) error {
	if code == "" {
		return fmt.Errorf("code is required")
	}
	if a.sidecar == nil {
		return fmt.Errorf("sidecar not running")
	}
	return a.sidecar.SubmitAuthCode(code)
}

// RestartSidecar stops and restarts the sidecar subprocess.
func (a *App) RestartSidecar() error {
	if a.sidecar != nil {
		a.sidecar.Stop()
	}
	settings, err := a.config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.sidecar = backend.NewDeezerSidecar("data", settings.PythonPath)
	if err := a.sidecar.Start(); err != nil {
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "sidecar:status", a.sidecar.Status())
	// Restart event stream.
	go func() {
		for ev := range a.sidecar.EventChan() {
			wailsruntime.EventsEmit(a.ctx, "sidecar:event", ev)
		}
	}()
	return nil
}

// TestSidecar sends a ping to verify the sidecar is responsive.
func (a *App) TestSidecar() (bool, error) {
	if a.sidecar == nil {
		return false, fmt.Errorf("sidecar not running")
	}
	_, err := a.sidecar.Search("test", "test")
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetSidecarCredentials stores Telegram API credentials in the OS keychain.
func (a *App) SetSidecarCredentials(apiID, apiHash, phone string) error {
	if apiID == "" || apiHash == "" || phone == "" {
		return fmt.Errorf("all credential fields are required")
	}
	return backend.SetSidecarCredentials(apiID, apiHash, phone)
}

// GetSidecarCredentials returns whether credentials are configured.
func (a *App) HasSidecarCredentials() bool {
	apiID, apiHash, phone := backend.GetSidecarCredentials()
	return apiID != "" && apiHash != "" && phone != ""
}

// SidecarDownload enqueues a download via the Deezer sidecar.
func (a *App) SidecarDownload(artist, title string) (map[string]interface{}, error) {
	if a.sidecar == nil {
		return nil, fmt.Errorf("sidecar not running")
	}
	// 1. Search for the track.
	search, err := a.sidecar.Search(artist, title)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	searchKey, _ := search["search_key"].(string)
	if searchKey == "" {
		return nil, fmt.Errorf("no results found")
	}
	// 2. Download the top result.
	taskID, err := a.sidecar.Download(searchKey, 0)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	return map[string]interface{}{
		"task_id":    taskID,
		"search_key": searchKey,
	}, nil
}

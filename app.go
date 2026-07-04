package main

import (
	"context"
	"errors"
	"strings"
	"fmt"
	"os"
	"path/filepath"
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
}

// shutdown is called by Wails before the window closes.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	a.started = false
	a.mu.Unlock()
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



// --- Dummy Stubs for SpotiFLAC React UI ---
func (a *App) CheckFFmpegInstalled() bool { return true }
func (a *App) DownloadFFmpeg() error { return nil }
func (a *App) GetDefaults() map[string]interface{} { return nil }
func (a *App) LoadFonts() map[string]interface{} { return nil }
func (a *App) LoadSettings() map[string]interface{} { return nil }
func (a *App) SaveFonts(f map[string]interface{}) error { return nil }
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

// GetSettings returns the current settings.
func (a *App) GetSettings() (backend.Settings, error) {
	return a.config.Load()
}

// SaveSettings persists settings.
func (a *App) SaveSettings(s backend.Settings) error {
	if err := backend.EnsureDownloadsFolder(s.DownloadsFolder); err != nil {
		return err
	}
	return a.config.Save(s)
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


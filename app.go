package main

import (
	"context"
	"errors"
	"strings"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
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
	sidecar *backend.Sidecar
	started bool

	// Durable copy of the most recent sidecar:status event so the
	// frontend can retrieve it via GetSidecarInfo even if it missed
	// the real-time event (race between EventsOn subscription and the
	// first EmitStatus).
	lastSidecarStatus  string
	lastSidecarMessage string
	lastSidecarLogs    []string
	lastPythonPath     string
	lastScriptPath     string
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

	go a.bootSidecar(ctx)
}

// bootSidecar launches (or re-launches) the Python sidecar and wires up
// its event handlers. Always emits a status event so the UI is updated
// even on failure.
func (a *App) bootSidecar(ctx context.Context) {
	logSidecar("bootSidecar: entered")
	defer func() {
		if r := recover(); r != nil {
			logSidecar(fmt.Sprintf("bootSidecar: PANIC: %v", r))
		}
		logSidecar("bootSidecar: leaving")
	}()

	// Stop any existing sidecar first (safe no-op if none running).
	a.mu.Lock()
	old := a.sidecar
	a.sidecar = nil
	a.mu.Unlock()
	if old != nil {
		old.Stop()
	}

	// Resolve the Python executable. If the user has set a path in
	// Settings, that takes priority; otherwise we auto-discover.
	pythonPath := a.resolvePythonPath()
	logSidecar(fmt.Sprintf("bootSidecar: python=%s", pythonPath))

	// Find the sidecar script.
	scriptPath, err := a.findSidecarScript()
	if err != nil {
		logSidecar(fmt.Sprintf("bootSidecar: findSidecarScript FAILED: %s", err.Error()))
		a.storeSidecarError("sidecar script not found: " + err.Error())
		backend.EmitStatus(ctx, backend.SidecarStatusError, err.Error())
		return
	}
	logSidecar(fmt.Sprintf("bootSidecar: script=%s", scriptPath))
	a.mu.Lock()
	a.lastScriptPath = scriptPath
	a.mu.Unlock()

	// Use the script's directory as the working directory so that
	// deezload.py can locate its sibling `.env` file.
	workDir := filepath.Dir(scriptPath)
	logSidecar(fmt.Sprintf("bootSidecar: workDir=%s", workDir))

	// Best-effort: if the script lives in the per-user extract dir
	// (production build) and the .env lives somewhere else, copy it
	// over so deezload.py's `load_dotenv()` finds it. We do not error
	// out if this fails — the user can drop the file there manually.
	extraEnv := a.copyDotEnvIfMissing(workDir)
	logSidecar(fmt.Sprintf("bootSidecar: extraEnv keys=%d", len(extraEnv)))

	sidecar, err := backend.NewSidecar(backend.SidecarConfig{
		PythonBinary: pythonPath,
		ScriptPath:   scriptPath,
		WorkDir:      workDir,
		ExtraEnv:     extraEnv,
	})
	if err != nil {
		logSidecar(fmt.Sprintf("bootSidecar: NewSidecar FAILED: %s", err.Error()))
		a.storeSidecarError("sidecar init: " + err.Error())
		backend.EmitStatus(ctx, backend.SidecarStatusError, "sidecar init: "+err.Error())
		return
	}
	a.mu.Lock()
	a.sidecar = sidecar
	a.mu.Unlock()

	handler := backend.NewSidecarEventHandler(ctx, a.tasks, a.history)

	// Wrap OnStatus so every change is also stored durably in the App
	// struct.  This lets GetSidecarInfo return the latest state even
	// when the frontend missed the real-time Wails event.
	sidecar.SetHandlers(handler.OnEvent, func(status, message string) {
		a.mu.Lock()
		a.lastSidecarStatus = status
		a.lastSidecarMessage = message
		a.lastSidecarLogs = sidecar.Logs()
		a.mu.Unlock()
		handler.OnStatus(status, message)
	})

	logSidecar("bootSidecar: calling sidecar.Start()...")
	if err := sidecar.Start(ctx); err != nil {
		logSidecar(fmt.Sprintf("bootSidecar: sidecar.Start FAILED: %s", err.Error()))
		a.storeSidecarError("sidecar start: " + err.Error())
		backend.EmitStatus(ctx, backend.SidecarStatusError, "sidecar start: "+err.Error())
		return
	}
	logSidecar("bootSidecar: sidecar.Start returned OK")

	// Forward current settings to the sidecar (downloads folder, etc.).
	go func() {
		s, err := a.config.Load()
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, _ = sidecar.Request(ctx, "set_settings", map[string]any{
			"downloads_folder": s.DownloadsFolder,
			"export_lrc_file":  s.ExportLrcFile,
		})
	}()
}

// logSidecar appends a single line to sidecar.log next to the executable.
// Best-effort: errors writing the log are silently ignored.
func logSidecar(msg string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(filepath.Dir(exe), "sidecar.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(time.Now().Format("2006-01-02 15:04:05.000") + " " + msg + "\n")
}

// resolvePythonPath returns the Python executable to use. It first checks
// the persisted Settings; if empty, falls back to auto-discovery.
func (a *App) resolvePythonPath() string {
	s, err := a.config.Load()
	if err == nil && s.PythonPath != "" {
		a.lastPythonPath = s.PythonPath
		return s.PythonPath
	}
	path, err := backend.FindPythonExecutable()
	if err != nil {
		a.lastPythonPath = "python" // UI fallback
		return "python"
	}
	a.lastPythonPath = path
	return path
}

// storeSidecarError records a durable error status and message.
func (a *App) storeSidecarError(msg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastSidecarStatus = backend.SidecarStatusError
	a.lastSidecarMessage = msg
}

// copyDotEnvIfMissing ensures the sidecar's workdir has a .env file.
// If the file already exists in workdir, nothing is done. Otherwise we
// look for one in a list of well-known locations and copy it over. We
// also parse it and return Telegram credentials as env overrides so the
// sidecar picks them up even if the copy step failed.
func (a *App) copyDotEnvIfMissing(workDir string) map[string]string {
	envPath := filepath.Join(workDir, ".env")
	// If workdir already has a .env, just load it and return the vars.
	if _, err := os.Stat(envPath); err == nil {
		kv, _ := backend.LoadDotEnv(envPath)
		return telegramEnv(kv)
	}
	// Otherwise, try to find a .env elsewhere and copy it in.
	src := backend.FindDotEnv()
	if src == "" {
		return nil
	}
	if data, err := os.ReadFile(src); err == nil {
		_ = os.WriteFile(envPath, data, 0o600)
	}
	kv, _ := backend.LoadDotEnv(src)
	return telegramEnv(kv)
}

// telegramEnv extracts the Telegram credentials from a .env map. The
// sidecar will use these in addition to (or in lieu of) its own dotenv
// lookup.
func telegramEnv(kv map[string]string) map[string]string {
	out := map[string]string{}
	for _, k := range []string{"TELEGRAM_API_ID", "TELEGRAM_API_HASH", "TELEGRAM_PHONE"} {
		if v, ok := kv[k]; ok && v != "" {
			out[k] = v
		}
	}
	return out
}

// shutdown is called by Wails before the window closes.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	s := a.sidecar
	a.sidecar = nil
	a.started = false
	a.mu.Unlock()
	if s != nil {
		s.Stop()
	}
	if a.storage != nil {
		_ = a.storage.Close()
	}
}

func (a *App) findSidecarScript() (string, error) {
	// On-disk candidates first. In dev mode (`wails dev`) the script
	// lives at `sidecar/deezload.py` relative to the project root and
	// changes are picked up on reload.
	candidates := []string{
		"sidecar/deezload.py",
		"./sidecar/deezload.py",
	}
	// Common locations relative to the running executable.
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// Same directory as the binary (post-build copy scenario):
		//   build/bin/Lymuru.exe  ->  build/bin/sidecar/deezload.py
		candidates = append(candidates, filepath.Join(dir, "sidecar", "deezload.py"))
		// One level up (project-layout scenario):
		//   build/bin/Lymuru.exe  ->  build/sidecar/deezload.py
		candidates = append(candidates, filepath.Join(dir, "..", "sidecar", "deezload.py"))
		// Two levels up (wails dev from project root):
		//   Lymuru/build/bin/Lymuru.exe  ->  Lymuru/sidecar/deezload.py
		candidates = append(candidates, filepath.Join(dir, "..", "..", "sidecar", "deezload.py"))
	}
	// User-wide install location (per-user, writable):
	if cfg, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(cfg, "Lymuru", "sidecar", "deezload.py"))
	}
	if local, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(local, ".Lymuru", "sidecar", "deezload.py"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs, nil
		}
	}
	// Fall back to extracting the script embedded in the binary. This
	// makes the production build self-contained when no sidecar is
	// shipped next to the executable.
	if path, err := backend.ExtractEmbeddedSidecar(); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("sidecar script not found in any of: %v", candidates)
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

// AuthState describes the current Telegram auth state.
type AuthState struct {
	State        string `json:"state"` // "authenticated" | "auth_required" | "unknown"
	Phone        string `json:"phone,omitempty"`
	SidecarReady bool   `json:"sidecar_ready"`
}

// GetVersion returns the app version.
func (a *App) GetVersion() string {
	return backend.AppVersion()
}

// GetSidecarStatus returns the current sidecar status, preferring the
// durable copy so late-arriving frontend subscribers still see the truth.
func (a *App) GetSidecarStatus() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastSidecarStatus != "" {
		return a.lastSidecarStatus
	}
	if a.sidecar != nil {
		return a.sidecar.Status()
	}
	return backend.SidecarStatusStopped
}

// GetSidecarLogs returns the most recent stderr lines emitted by the
// sidecar. Useful for diagnosing startup failures (e.g. missing
// Python dependencies) without having to dig through the OS logs.
func (a *App) GetSidecarLogs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sidecar != nil {
		return a.sidecar.Logs()
	}
	return a.lastSidecarLogs
}

// GetSidecarInfo returns a bundle of sidecar diagnostic info
// (status, message, recent log lines, Python path, script dir).
type SidecarInfo struct {
	Status     string   `json:"status"`
	Message    string   `json:"message"`
	Logs       []string `json:"logs"`
	ScriptDir  string   `json:"script_dir"`
	PythonPath string   `json:"python_path"`
}

func (a *App) GetSidecarInfo() SidecarInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	info := SidecarInfo{
		PythonPath: a.lastPythonPath,
		ScriptDir:  a.lastScriptPath,
	}
	// Prefer the live sidecar snapshot over the durable fields.
	if a.sidecar != nil {
		info.Status, info.Message, info.Logs = a.sidecar.Snapshot()
		info.ScriptDir = filepath.Dir(a.sidecar.ScriptPath())
	} else {
		info.Status = a.lastSidecarStatus
		info.Message = a.lastSidecarMessage
		info.Logs = append([]string{}, a.lastSidecarLogs...)
	}
	if info.Status == "" {
		info.Status = backend.SidecarStatusStopped
	}
	if info.PythonPath == "" {
		info.PythonPath = "python"
	}
	return info
}

// TestSidecar kicks off a sidecar restart and returns immediately. The
// frontend should poll GetSidecarInfo() to observe the result.
type TestSidecarResponse struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (a *App) TestSidecar() TestSidecarResponse {
	if a.ctx == nil {
		return TestSidecarResponse{OK: false, Status: "error", Message: "app not ready"}
	}
	logSidecar("TestSidecar: triggered")
	go a.bootSidecar(a.ctx)
	// Return immediately; the frontend polls GetSidecarInfo.
	return TestSidecarResponse{OK: true, Status: "starting", Message: "Sidecar restart initiated"}
}

// GetAuthState returns whether the user is authenticated with Telegram.
func (a *App) GetAuthState() AuthState {
	a.mu.Lock()
	s := a.sidecar
	a.mu.Unlock()
	state := AuthState{State: "unknown"}
	if s == nil {
		return state
	}
	switch s.Status() {
	case backend.SidecarStatusAuth:
		state.State = "auth_required"
		// Phone is included in the status event; we can't easily get it here,
		// so leave empty and let the sidecar emit auth:needed with the phone.
	case backend.SidecarStatusOnline:
		state.State = "authenticated"
	case backend.SidecarStatusStarting, backend.SidecarStatusStopped:
		state.State = "unknown"
	case backend.SidecarStatusError:
		state.State = "error"
	}
	state.SidecarReady = s.Status() != backend.SidecarStatusStopped
	return state
}

// SubmitAuthCode forwards a Telegram verification code to the sidecar.
func (a *App) SubmitAuthCode(code string) error {
	a.mu.Lock()
	s := a.sidecar
	a.mu.Unlock()
	if s == nil {
		return errors.New("sidecar not running")
	}
	// Use a short timeout so the UI never freezes if the sidecar
	// doesn't respond (e.g. because another handler blocked stdin).
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	if _, err := s.Request(ctx, "submit_auth", map[string]any{"code": code}); err != nil {
		return err
	}
	return nil
}

// SignOut is a no-op for now; the session file remains on disk. Users can
// delete it manually if they want a clean re-auth.
func (a *App) SignOut() error { return nil }

// RestartSidecar stops the running sidecar (if any) and starts a fresh
// one. The new status is emitted via the sidecar:status event.
func (a *App) RestartSidecar() error {
	if a.ctx == nil {
		return errors.New("app not ready")
	}
	go a.bootSidecar(a.ctx)
	return nil
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

func (a *App) simpleTask(method string, params map[string]any, query string) (SidecarTaskResponse, error) {
	if a.ctx == nil {
		return SidecarTaskResponse{}, errors.New("app not ready")
	}
	s := a.getSidecar()
	if s == nil {
		return SidecarTaskResponse{}, errors.New("sidecar not running")
	}
	taskID := uuid.NewString()
	a.tasks.Add(backend.Task{
		TaskID:    taskID,
		TaskType:  method,
		Query:     query,
		Stage:     "Queued",
		Phase:     "preparing",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if params == nil {
		params = map[string]any{}
	}
	params["task_id"] = taskID
	resp, err := s.Request(a.ctx, method, params)
	if err != nil {
		a.tasks.Remove(taskID)
		return SidecarTaskResponse{}, err
	}
	if !resp.OK {
		a.tasks.Remove(taskID)
		return SidecarTaskResponse{}, errors.New(resp.Error)
	}
	return SidecarTaskResponse{TaskID: taskID}, nil
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

func (a *App) getSidecar() *backend.Sidecar {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sidecar
}

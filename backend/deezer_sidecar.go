package backend

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// SidecarCommand is a JSON-RPC request sent to the Python sidecar on stdin.
type SidecarCommand struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// sidecarResponse is a JSON-RPC response read from the sidecar's stdout.
type sidecarResponse struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// SidecarEvent is an async JSON event read from the sidecar's stderr.
type SidecarEvent struct {
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	TaskID          string   `json:"task_id"`
	Stage           string   `json:"stage,omitempty"`
	Phase           string   `json:"phase,omitempty"`
	DownloadPercent float64  `json:"download_percent,omitempty"`
	Files           []string `json:"files,omitempty"`
	Message         string   `json:"message,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// SidecarStatus holds public-facing sidecar state.
type SidecarStatus struct {
	Running       bool   `json:"running"`
	Authenticated bool   `json:"authenticated"`
	Error         string `json:"error,omitempty"`
}

// SidecarDownloadRequest is used internally to track a download task.
type SidecarDownloadRequest struct {
	TaskID    string
	Artist    string
	Title     string
	OutputDir string
	Response  chan SidecarDownloadResult
}

// SidecarDownloadResult is the result of a sidecar download.
type SidecarDownloadResult struct {
	FilePath string
	Error    string
}

// pending tracks a request waiting for a JSON-RPC response.
type pendingRequest struct {
	ch      chan sidecarResponse
	timeout time.Time
}

// ---------------------------------------------------------------------------
// DeezerSidecar
// ---------------------------------------------------------------------------

// DeezerSidecar manages the Python sidecar subprocess for Deezer downloads.
// Communication follows a JSON-RPC protocol over stdin/stdout, with async
// events on stderr.
type DeezerSidecar struct {
	mu            sync.Mutex
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	running       bool
	authenticated bool
	startErr      string
	dataDir       string

	pending   map[string]pendingRequest
	reqSeq    int
	events    chan SidecarEvent
	eventDone chan struct{}
	respDone  chan struct{}

	// Credentials (loaded from keychain at init).
	apiID   string
	apiHash string
	phone   string

	// Python path.
	pythonPath string

	// Exit monitoring.
	onStatusChange func(SidecarStatus)
	stderrBuf      strings.Builder // captured stderr for diagnostics on crash
}

// NewDeezerSidecar creates a new sidecar manager. Call Start() to spawn the
// Python subprocess.
func NewDeezerSidecar(dataDir, pythonPath string) *DeezerSidecar {
	apiID, apiHash, phone := GetSidecarCredentials()
	return &DeezerSidecar{
		dataDir:    dataDir,
		pythonPath: pythonPath,
		apiID:      apiID,
		apiHash:    apiHash,
		phone:      phone,
		events:     make(chan SidecarEvent, 64),
	}
}

// HasCredentials returns true if all Telegram credentials are configured.
func (s *DeezerSidecar) HasCredentials() bool {
	return s.apiID != "" && s.apiHash != "" && s.phone != ""
}

// Start spawns the Python sidecar process.
func (s *DeezerSidecar) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	if !s.HasCredentials() {
		s.startErr = "Telegram credentials not configured"
		return fmt.Errorf("%s", s.startErr)
	}

	// Reset start error and stderr buffer.
	s.startErr = ""
	s.stderrBuf.Reset()

	// Resolve the Python executable.
	pyPath := s.pythonPath
	if pyPath == "" {
		pyPath = resolvePython()
	}
	if pyPath == "" {
		s.startErr = "python not found"
		return fmt.Errorf("%s", s.startErr)
	}

	// Resolve the sidecar script path.
	sidecarPath := filepath.Join("sidecar", "deezload.py")

	// Build the command.
	s.cmd = exec.Command(pyPath, sidecarPath, "--sidecar")
	s.cmd.Dir = s.dataDir

	// Pass credentials as environment variables.
	s.cmd.Env = append(os.Environ(),
		"TELEGRAM_API_ID="+s.apiID,
		"TELEGRAM_API_HASH="+s.apiHash,
		"TELEGRAM_PHONE="+s.phone,
		"LYMURU_DOWNLOAD_DIR="+filepath.Join(s.dataDir, "downloads"),
	)

	// Stdin pipe for sending commands.
	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		s.startErr = err.Error()
		return fmt.Errorf("sidecar stdin: %w", err)
	}

	// Stdout pipe for reading JSON-RPC responses.
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		s.startErr = err.Error()
		return fmt.Errorf("sidecar stdout: %w", err)
	}

	// Stderr pipe for reading async events.
	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		s.startErr = err.Error()
		return fmt.Errorf("sidecar stderr: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		s.startErr = err.Error()
		return fmt.Errorf("sidecar start: %w", err)
	}

	s.running = true
	s.pending = make(map[string]pendingRequest)
	s.reqSeq = 0
	s.eventDone = make(chan struct{})
	s.respDone = make(chan struct{})

	// Read JSON-RPC responses from stdout.
	go func() {
		defer close(s.respDone)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			var resp sidecarResponse
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				LogWarn("[Sidecar] bad stdout line: %v", err)
				continue
			}
			s.mu.Lock()
			pr, ok := s.pending[resp.ID]
			if ok {
				delete(s.pending, resp.ID)
			}
			s.mu.Unlock()
			if ok {
				select {
				case pr.ch <- resp:
				default:
				}
			}
		}
	}()

	// Read async events from stderr.
	go func() {
		defer close(s.eventDone)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Capture last lines of stderr for crash diagnostics.
			s.appendStderr(line)
			var ev SidecarEvent
			if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
				LogDebug("[Sidecar] non-event stderr: %s", line)
				continue
			}
			if ev.Name == "auth_needed" {
				s.mu.Lock()
				s.authenticated = false
				s.mu.Unlock()
			}
			select {
			case s.events <- ev:
			default:
			}
		}
	}()

	// Wait briefly for startup, then ping to confirm readiness.
	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := s.call("ping", nil, 5*time.Second); err != nil {
			LogWarn("[Sidecar] startup ping failed: %v", err)
		} else {
			LogInfo("[Sidecar] started successfully")
		}
	}()

	// Monitor process exit. When the process dies, update state and notify.
	go func() {
		waitErr := s.cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.authenticated = false
		if waitErr != nil {
			s.startErr = waitErr.Error()
		}
		// Append captured stderr for better diagnostics.
		stderrText := strings.TrimSpace(s.stderrBuf.String())
		stderrLower := strings.ToLower(stderrText)
		if strings.Contains(stderrLower, "microsoft store") || strings.Contains(stderrLower, "python was not found") {
			s.startErr = "Python is not installed or only the Microsoft Store alias is available. Install Python from python.org and restart, or set the Python Path in Settings → Deezer."
		} else if stderrText != "" {
			lastLine := stderrText
			if idx := strings.LastIndex(lastLine, "\n"); idx >= 0 {
				lastLine = lastLine[idx+1:]
			}
			if s.startErr == "" {
				s.startErr = lastLine
			} else {
				s.startErr = fmt.Sprintf("%s (stderr: %s)", s.startErr, lastLine)
			}
		}
		// Unblock pending callers.
		for id, pr := range s.pending {
			select {
			case pr.ch <- sidecarResponse{ID: id, OK: false, Error: "sidecar process exited"}:
			default:
			}
			delete(s.pending, id)
		}
		status := SidecarStatus{
			Running:       s.running,
			Authenticated: s.authenticated,
			Error:         s.startErr,
		}
		cb := s.onStatusChange
		s.mu.Unlock()
		if cb != nil {
			cb(status)
		}
	}()

	return nil
}

// Stop gracefully shuts down the sidecar.
func (s *DeezerSidecar) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	// Send shutdown command (fire-and-forget).
	cmd := SidecarCommand{ID: "shutdown", Method: "shutdown", Params: map[string]interface{}{}}
	if s.stdin != nil {
		raw, _ := json.Marshal(cmd)
		_, _ = fmt.Fprintln(s.stdin, string(raw))
	}

	// Wait briefly, then kill.
	time.Sleep(200 * time.Millisecond)
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	s.running = false
	s.authenticated = false

	// Close the events channel so the forwarding goroutine exits.
	close(s.events)
}

// IsRunning returns true if the sidecar process is alive.
func (s *DeezerSidecar) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// IsAuthenticated returns true if the Telegram session is valid.
func (s *DeezerSidecar) IsAuthenticated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.authenticated
}

// EventChan returns the channel for async sidecar events.
func (s *DeezerSidecar) EventChan() <-chan SidecarEvent {
	return s.events
}

// Status returns the current sidecar status.
func (s *DeezerSidecar) Status() SidecarStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SidecarStatus{
		Running:       s.running,
		Authenticated: s.authenticated,
		Error:         s.startErr,
	}
}

// SetOnStatusChange registers a callback invoked when the sidecar status
// changes (e.g., on process exit). The callback fires under no lock.
func (s *DeezerSidecar) SetOnStatusChange(cb func(SidecarStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onStatusChange = cb
}

// SubmitAuthCode forwards an auth code to the sidecar.
func (s *DeezerSidecar) SubmitAuthCode(code string) error {
	_, err := s.call("submit_auth", map[string]interface{}{"code": code}, 30*time.Second)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.authenticated = true
	s.mu.Unlock()
	return nil
}

// Search looks up a track on Deezer via Telegram.
func (s *DeezerSidecar) Search(artist, title string) (map[string]interface{}, error) {
	resp, err := s.call("search", map[string]interface{}{
		"artist": artist,
		"title":  title,
	}, 60*time.Second)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	return result, nil
}

// Download enqueues a Deezer download via the sidecar.
// Returns the task_id for progress tracking.
func (s *DeezerSidecar) Download(searchKey string, choice int) (string, error) {
	resp, err := s.call("download", map[string]interface{}{
		"search_key": searchKey,
		"choice":     choice,
	}, 300*time.Second)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parse download response: %w", err)
	}
	taskID, _ := result["task_id"].(string)
	return taskID, nil
}

// DownloadLink downloads from a direct Deezer URL.
func (s *DeezerSidecar) DownloadLink(url string) (string, error) {
	resp, err := s.call("download_link", map[string]interface{}{
		"url": url,
	}, 300*time.Second)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parse download response: %w", err)
	}
	taskID, _ := result["task_id"].(string)
	return taskID, nil
}

// SetSettings sends updated settings to the sidecar.
func (s *DeezerSidecar) SetSettings(downloadsFolder string, exportLrc bool) error {
	_, err := s.call("set_settings", map[string]interface{}{
		"downloads_folder": downloadsFolder,
		"export_lrc_file":  exportLrc,
	}, 5*time.Second)
	return err
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// appendStderr appends a line to the stderr buffer, keeping only the last 4 KB.
func (s *DeezerSidecar) appendStderr(line string) {
	s.stderrBuf.WriteString(line)
	s.stderrBuf.WriteByte('\n')
	// Trim to last ~4 KB to avoid unbounded growth.
	if s.stderrBuf.Len() > 4096 {
		buf := s.stderrBuf.String()
		keep := buf[len(buf)-4096:]
		s.stderrBuf.Reset()
		s.stderrBuf.WriteString(keep)
	}
}

func (s *DeezerSidecar) call(method string, params map[string]interface{}, timeout time.Duration) (sidecarResponse, error) {
	if params == nil {
		params = map[string]interface{}{}
	}

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return sidecarResponse{}, fmt.Errorf("sidecar not running")
	}
	s.reqSeq++
	id := fmt.Sprintf("%s-%d", uuid.New().String()[:8], s.reqSeq)
	ch := make(chan sidecarResponse, 1)
	s.pending[id] = pendingRequest{ch: ch, timeout: time.Now().Add(timeout)}
	s.mu.Unlock()

	cmd := SidecarCommand{ID: id, Method: method, Params: params}
	raw, err := json.Marshal(cmd)
	if err != nil {
		return sidecarResponse{}, fmt.Errorf("marshal command: %w", err)
	}

	if _, err := fmt.Fprintln(s.stdin, string(raw)); err != nil {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return sidecarResponse{}, fmt.Errorf("write command: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-ch:
		if !resp.OK {
			return resp, fmt.Errorf("sidecar error: %s", resp.Error)
		}
		return resp, nil
	case <-timer.C:
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return sidecarResponse{}, fmt.Errorf("sidecar request timed out after %v", timeout)
	}
}

// resolvePython finds a working Python 3 executable on the system PATH.
// On Windows it also tries the "py" launcher and verifies candidates are
// not the Microsoft Store stub.
func resolvePython() string {
	for _, name := range []string{"python3", "python", "py"} {
		p, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		// Quick verification: run python --version to confirm it's real.
		if isRealPython(p) {
			return p
		}
	}
	return ""
}

// isRealPython runs the given executable with --version and checks if it
// outputs a Python version string (not a Microsoft Store alias message).
func isRealPython(exe string) bool {
	out, err := exec.Command(exe, "--version").CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	return strings.Contains(s, "python")
}

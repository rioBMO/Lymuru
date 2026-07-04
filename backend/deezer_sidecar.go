package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// SidecarStatus values for the UI.
const (
	SidecarStatusStopped   = "stopped"
	SidecarStatusStarting  = "starting"
	SidecarStatusOnline    = "online"
	SidecarStatusAuth      = "auth_required"
	SidecarStatusError     = "error"
)

// SidecarEvent is emitted by the Python sidecar on stderr (JSON lines).
type SidecarEvent struct {
	Type    string          `json:"type"`
	Name    string          `json:"name,omitempty"`
	TaskID  string          `json:"task_id,omitempty"`
	Stage   string          `json:"stage,omitempty"`
	Phase   string          `json:"phase,omitempty"`
	Percent float64         `json:"download_percent,omitempty"`
	Files   []string        `json:"files,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
	Phone   string          `json:"phone,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// SidecarRequest is a JSON line sent to the sidecar on stdin.
type SidecarRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// SidecarResponse is a JSON line read from the sidecar's stdout.
type SidecarResponse struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Sidecar owns the deezload.py subprocess and the JSON-RPC client.
type Sidecar struct {
	pythonPath string
	scriptPath string
	workDir    string
	extraEnv   map[string]string

	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	pyStderrFile *os.File
	stderr   *bufio.Reader
	pending  map[string]chan SidecarResponse
	cancelF  map[string]context.CancelFunc
	closed   atomic.Bool
	status   string
	lastErr  string

	// Ring buffer of recent stderr lines for diagnostics. When the
	// subprocess dies or ping fails, the last lines are surfaced in the
	// sidecar:status event so the user can see what went wrong.
	logBuf []string

	// Channel for sending the auth code to the running sidecar.
	authCh chan string

	// OnEvent is called for each event line parsed from stderr.
	OnEvent func(SidecarEvent)
	// OnStatus is called whenever the sidecar status changes.
	OnStatus func(status, message string)
}

const sidecarLogBufferSize = 50

// SidecarConfig configures a new Sidecar.
type SidecarConfig struct {
	PythonBinary string            // path to python; defaults to "python" / "python3"
	ScriptPath   string            // path to deezload.py
	WorkDir      string            // sidecar's working directory (where .env lives)
	ExtraEnv     map[string]string // additional env vars to pass to the subprocess
}

// NewSidecar constructs a new Sidecar. It does not start the subprocess.
func NewSidecar(cfg SidecarConfig) (*Sidecar, error) {
	if cfg.ScriptPath == "" {
		return nil, errors.New("sidecar: ScriptPath required")
	}
	if cfg.PythonBinary == "" {
		if runtime.GOOS == "windows" {
			cfg.PythonBinary = "python"
		} else {
			cfg.PythonBinary = "python3"
		}
	}
	if _, err := os.Stat(cfg.ScriptPath); err != nil {
		return nil, fmt.Errorf("sidecar: script not found: %w", err)
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = filepath.Dir(cfg.ScriptPath)
	}
	return &Sidecar{
		pythonPath: cfg.PythonBinary,
		scriptPath: cfg.ScriptPath,
		workDir:    cfg.WorkDir,
		extraEnv:   cfg.ExtraEnv,
		pending:    map[string]chan SidecarResponse{},
		cancelF:    map[string]context.CancelFunc{},
		authCh:     make(chan string, 1),
		status:     SidecarStatusStopped,
		logBuf:     make([]string, 0, sidecarLogBufferSize),
	}, nil
}

// SetHandlers sets the event and status callbacks.
func (s *Sidecar) SetHandlers(onEvent func(SidecarEvent), onStatus func(string, string)) {
	s.OnEvent = onEvent
	s.OnStatus = onStatus
}

// Status returns the current sidecar status string.
func (s *Sidecar) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// ScriptPath returns the path to the Python script managed by this sidecar.
func (s *Sidecar) ScriptPath() string { return s.scriptPath }

// Start launches the sidecar subprocess and starts the read loops.
func (s *Sidecar) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.cmd != nil {
		s.mu.Unlock()
		return errors.New("sidecar: already running")
	}
	// Set the status directly while holding the lock to avoid a
	// re-entrant deadlock (setStatus also acquires s.mu).
	s.status = SidecarStatusStarting
	s.lastErr = ""
	s.mu.Unlock()
	// Fire the status callback outside the lock.
	if s.OnStatus != nil {
		s.OnStatus(SidecarStatusStarting, "")
	}

	dbgLog("Start: pythonPath=%s scriptPath=%s workDir=%s", s.pythonPath, s.scriptPath, s.workDir)
	dbgLog("Start: creating command...")
	cmd := exec.CommandContext(ctx, s.pythonPath, s.scriptPath, "--sidecar")
	cmd.Dir = s.workDir
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	for k, v := range s.extraEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// Open a log file for Python's stderr so crashes are preserved,
	// but always route stderr through a pipe so we can parse events
	// (auth_needed, progress, etc.) in real time.
	pyStderrFile, pyStderrPath := (*os.File)(nil), ""
	if exe, err := os.Executable(); err == nil {
		pyStderrPath = filepath.Join(filepath.Dir(exe), "python-stderr.log")
		if f, err := os.Create(pyStderrPath); err == nil {
			pyStderrFile = f
			dbgLog("Start: python stderr will be mirrored to %s", pyStderrPath)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		dbgLog("Start: stdin pipe FAILED: %v", err)
		if pyStderrFile != nil {
			pyStderrFile.Close()
		}
		return fmt.Errorf("sidecar: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		dbgLog("Start: stdout pipe FAILED: %v", err)
		stdin.Close()
		if pyStderrFile != nil {
			pyStderrFile.Close()
		}
		return fmt.Errorf("sidecar: stdout pipe: %w", err)
	}
	// Always use StderrPipe — we need to read events in real time.
	// The log file is written to manually inside readStderrLoop.
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		dbgLog("Start: stderr pipe FAILED: %v", err)
		stdin.Close()
		stdout.Close()
		if pyStderrFile != nil {
			pyStderrFile.Close()
		}
		return fmt.Errorf("sidecar: stderr pipe: %w", err)
	}
	s.stderr = bufio.NewReader(stderrPipe)
	dbgLog("Start: calling cmd.Start()...")
	if err := cmd.Start(); err != nil {
		dbgLog("Start: cmd.Start FAILED: %v", err)
		stdin.Close()
		stdout.Close()
		if pyStderrFile != nil {
			pyStderrFile.Close()
		}
		return fmt.Errorf("sidecar: start: %w", err)
	}
	dbgLog("Start: cmd.Start OK, PID=%d", cmd.Process.Pid)

	// Re-acquire the lock to publish the subprocess fields. The
	// initial lock was released above so OnStatus could fire without
	// re-entering setStatus; now that pipes are open and the process
	// is running, publish the handles atomically.
	s.mu.Lock()
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReader(stdout)
	// The log file is written to by readStderrLoop; waitLoop closes it on exit.
	s.pyStderrFile = pyStderrFile
	s.closed.Store(false)
	s.mu.Unlock()

	go s.readStdoutLoop()
	// s.stderr is always set (StderrPipe), so readStderrLoop always runs.
	go s.readStderrLoop()
	go s.waitLoop()

	dbgLog("Start: entering pingLoop (30s timeout)...")
	if err := s.pingLoop(ctx); err != nil {
		dbgLog("Start: pingLoop FAILED: %v", err)
		s.mu.Lock()
		if s.status == SidecarStatusStarting {
			s.setStatus(SidecarStatusError, err.Error())
		}
		s.mu.Unlock()
		return err
	}
	dbgLog("Start: pingLoop OK, sidecar online")
	s.setStatus(SidecarStatusOnline, "")
	return nil
}

// Stop terminates the sidecar subprocess.
func (s *Sidecar) Stop() {
	s.closed.Store(true)
	s.mu.Lock()
	if s.cmd == nil || s.cmd.Process == nil {
		s.mu.Unlock()
		return
	}
	cmd := s.cmd
	if s.stdin != nil {
		_ = s.stdin.Close()
		s.stdin = nil
	}
	s.mu.Unlock()
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	s.setStatus(SidecarStatusStopped, "")
}

// Request sends a JSON-RPC request to the sidecar and waits for the response.
func (s *Sidecar) Request(ctx context.Context, method string, params any) (SidecarResponse, error) {
	s.mu.Lock()
	if s.cmd == nil || s.stdin == nil {
		s.mu.Unlock()
		return SidecarResponse{}, errors.New("sidecar: not running")
	}
	id := uuid.NewString()
	req := SidecarRequest{ID: id, Method: method}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			s.mu.Unlock()
			return SidecarResponse{}, err
		}
		req.Params = b
	}
	ch := make(chan SidecarResponse, 1)
	s.pending[id] = ch
	stdin := s.stdin
	s.mu.Unlock()

	b, _ := json.Marshal(req)
	b = append(b, '\n')
	if _, err := stdin.Write(b); err != nil {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return SidecarResponse{}, fmt.Errorf("sidecar: write: %w", err)
	}
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return SidecarResponse{}, ctx.Err()
	}
}

// SubmitAuthCode sends the Telegram verification code to the sidecar.
func (s *Sidecar) SubmitAuthCode(code string) error {
	select {
	case s.authCh <- code:
		return nil
	default:
		return errors.New("sidecar: auth channel full")
	}
}

func (s *Sidecar) setStatus(status, message string) {
	s.mu.Lock()
	s.status = status
	s.lastErr = message
	s.mu.Unlock()
	if s.OnStatus != nil {
		s.OnStatus(status, message)
	}
}

func (s *Sidecar) readStdoutLoop() {
	for {
		s.mu.Lock()
		stdout := s.stdout
		closed := s.closed.Load()
		s.mu.Unlock()
		if closed || stdout == nil {
			return
		}
		line, err := stdout.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.setStatus(SidecarStatusError, "stdout read: "+err.Error())
			}
			return
		}
		var resp SidecarResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			// Skip malformed lines.
			continue
		}
		s.mu.Lock()
		ch, ok := s.pending[resp.ID]
		if ok {
			delete(s.pending, resp.ID)
		}
		s.mu.Unlock()
		if ok {
			ch <- resp
		}
	}
}

func (s *Sidecar) readStderrLoop() {
	for {
		s.mu.Lock()
		stderr := s.stderr
		pyStderrFile := s.pyStderrFile
		closed := s.closed.Load()
		s.mu.Unlock()
		if closed || stderr == nil {
			return
		}
		line, err := stderr.ReadString('\n')
		if err != nil {
			return
		}
		line = trimNewline(line)
		if line == "" {
			continue
		}
		// Mirror each line to the log file (best-effort; ignore write errors).
		if pyStderrFile != nil {
			_, _ = pyStderrFile.WriteString(line + "\n")
		}
		log.Printf("sidecar stderr: %s", line)
		s.appendLog(line)
		var ev SidecarEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			// Treat as a plain log line; surface as a generic event.
			ev = SidecarEvent{Type: "log", Message: line}
		}
		if s.OnEvent != nil {
			s.OnEvent(ev)
		}
		switch ev.Name {
		case "auth_needed":
			s.setStatus(SidecarStatusAuth, ev.Phone)
		case "auth_success":
			s.setStatus(SidecarStatusOnline, "")
		}
	}
}

// appendLog stores the line in the ring buffer for later diagnostics.
func (s *Sidecar) appendLog(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.logBuf) >= sidecarLogBufferSize {
		// Drop the oldest entry.
		s.logBuf = s.logBuf[1:]
	}
	s.logBuf = append(s.logBuf, line)
}

// Logs returns a copy of the most recent stderr lines.
func (s *Sidecar) Logs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.logBuf))
	copy(out, s.logBuf)
	return out
}

// Snapshot returns the current status, last error message, and recent
// logs as a single consistent read.
func (s *Sidecar) Snapshot() (status, message string, logs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status = s.status
	message = s.lastErr
	logs = make([]string, len(s.logBuf))
	copy(logs, s.logBuf)
	return
}

func (s *Sidecar) waitLoop() {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd == nil {
		log.Printf("waitLoop: cmd is nil, exiting")
		return
	}
	log.Printf("waitLoop: waiting for process...")
	err := cmd.Wait()
	log.Printf("waitLoop: process exited, err=%v", err)
	s.mu.Lock()
	s.cmd = nil
	s.stdin = nil
	s.stdout = nil
	s.stderr = nil
	// Close the python stderr file if we created one.
	if s.pyStderrFile != nil {
		_ = s.pyStderrFile.Close()
		s.pyStderrFile = nil
	}
	// Close any pending requests.
	for id, ch := range s.pending {
		ch <- SidecarResponse{ID: id, OK: false, Error: "sidecar exited"}
		delete(s.pending, id)
	}
	s.mu.Unlock()
	if s.closed.Load() {
		return
	}
	msg := "sidecar exited"
	if err != nil {
		msg = "sidecar exited: " + err.Error()
	}
	// Surface the last few stderr lines so the user can see what
	// crashed (e.g. "ModuleNotFoundError: No module named 'telethon'").
	if logs := s.Logs(); len(logs) > 0 {
		tail := logs
		if len(tail) > 3 {
			tail = tail[len(tail)-3:]
		}
		msg += " — " + strings.Join(tail, " | ")
	}
	s.setStatus(SidecarStatusError, msg)
}

func (s *Sidecar) pingLoop(ctx context.Context) error {
	dbgLog("pingLoop: entering, sending ping request")
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := s.Request(deadline, "ping", nil)
	dbgLog("pingLoop: Request returned, err=%v ok=%v", err, resp.OK)
	if err != nil {
		// On timeout, include the most recent stderr lines so the
		// user can see whether Python is still booting up or
		// already crashed.
		if deadline.Err() != nil {
			if logs := s.Logs(); len(logs) > 0 {
				tail := logs
				if len(tail) > 3 {
					tail = tail[len(tail)-3:]
				}
				return fmt.Errorf("sidecar did not respond to ping within 30s — last log: %s",
					strings.Join(tail, " | "))
			}
		}
		return err
	}
	if !resp.OK {
		return errors.New("sidecar ping failed: " + resp.Error)
	}
	return nil
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func dbgLog(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := time.Now().Format("15:04:05.000") + " sidecar:" + msg + "\n"

	exe, err := os.Executable()
	if err != nil {
		os.Stderr.WriteString("dbgLog: os.Executable failed: " + err.Error() + "\n")
		return
	}
	dir := filepath.Dir(exe)
	// Write to both boot.log (legacy) and sidecar.log (current).
	for _, name := range []string{"boot.log", "sidecar.log"} {
		f, err := os.OpenFile(filepath.Join(dir, name),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			continue
		}
		f.WriteString(line)
		f.Close()
	}
}

package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logMu sync.Mutex
	logFh *os.File
)

// InitLogger opens the log file for writing. Safe to call multiple times;
// the file is opened once. Pass the data directory (e.g. "data").
func InitLogger(dataDir string) error {
	logMu.Lock()
	defer logMu.Unlock()
	if logFh != nil {
		return nil
	}
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(logDir, "lymuru.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	logFh = f
	return nil
}

// LogDebug writes a debug-level message to the log file.
func LogDebug(format string, args ...interface{}) { writeLog("DEBUG", format, args...) }

// LogInfo writes an info-level message to the log file.
func LogInfo(format string, args ...interface{}) { writeLog("INFO", format, args...) }

// LogWarn writes a warning-level message to the log file.
func LogWarn(format string, args ...interface{}) { writeLog("WARN", format, args...) }

// LogError writes an error-level message to the log file.
func LogError(format string, args ...interface{}) { writeLog("ERROR", format, args...) }

func writeLog(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), level, msg)

	logMu.Lock()
	defer logMu.Unlock()

	// Always write to log file.
	if logFh != nil {
		_, _ = logFh.WriteString(line)
	}

	// Also echo to stdout so existing workflows keep working.
	fmt.Print(line)
}

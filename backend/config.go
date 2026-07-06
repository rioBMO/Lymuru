package backend

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/lymuru/lymuru/backend/storage"
)

// Settings holds the persisted app settings.
type Settings struct {
	ThemeMode              string `json:"theme_mode"`               // "light" or "dark"
	DownloadsFolder        string `json:"downloads_folder"`         // absolute path
	HasCompletedOnboarding bool   `json:"has_completed_onboarding"` // onboarding done
	ExportLrcFile          bool   `json:"export_lrc_file"`          // save .lrc file alongside downloaded audio
	FFmpegPath             string `json:"ffmpeg_path"`              // path to ffmpeg executable; auto-detect if empty
	AudioSource            string `json:"audio_source"`             // "auto" (default), "tidal", "amazon", "qobuz", or "deezer"

	AudioFormat           string `json:"audio_format"`
	FilenameFormat        string `json:"filename_format"`
	CustomTidalAPI        string `json:"custom_tidal_api"`
	CustomQobuzAPI        string `json:"custom_qobuz_api"`
	ExistingFileCheckMode string `json:"existing_file_check_mode"`
	LinkResolver          string `json:"link_resolver"`
	AutoOrder             string `json:"auto_order"`
	Separator             string `json:"separator"`
	// Sidecar / Deezer settings.
	SidecarEnabled bool   `json:"sidecar_enabled"`
	PythonPath     string `json:"python_path,omitempty"`
}

// DefaultSettings returns sensible defaults. DownloadsFolder is expanded.
func DefaultSettings() Settings {
	home, _ := os.UserHomeDir()
	dl := filepath.Join(home, "Music", "Lymuru")
	return Settings{
		ThemeMode:              "light",
		DownloadsFolder:        dl,
		HasCompletedOnboarding: false,
		ExportLrcFile:          true,
		AudioSource:            "auto",
		AudioFormat:            "LOSSLESS",
		FilenameFormat:         "title-artist",
		ExistingFileCheckMode:  "filename",
		LinkResolver:           "deezer-songlink",
		AutoOrder:              "tidal-qobuz-amazon",
		Separator:              "comma",
	}
}

// Config owns the settings table.
type Config struct {
	mu sync.RWMutex
	db *storage.DB
}

func NewConfig(db *storage.DB) *Config { return &Config{db: db} }

// Load returns the persisted settings merged with defaults.
func (c *Config) Load() (Settings, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := DefaultSettings()
	if c.db == nil {
		return out, nil
	}
	rows, err := c.db.Conn().Query(`SELECT key, value FROM settings`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "theme_mode":
			out.ThemeMode = v
		case "downloads_folder":
			out.DownloadsFolder = v
		case "has_completed_onboarding":
			out.HasCompletedOnboarding = v == "1" || v == "true"
		case "export_lrc_file":
			out.ExportLrcFile = v == "1" || v == "true"
		case "ffmpeg_path":
			out.FFmpegPath = v
		case "audio_source":
			if v != "" {
				out.AudioSource = v
			}
		case "audio_format":
			out.AudioFormat = v
		case "filename_format":
			out.FilenameFormat = v
		case "custom_tidal_api":
			out.CustomTidalAPI = v
		case "custom_qobuz_api":
			out.CustomQobuzAPI = v
		case "existing_file_check_mode":
			out.ExistingFileCheckMode = v
		case "link_resolver":
			out.LinkResolver = v
		case "auto_order":
			out.AutoOrder = v
		case "separator":
			out.Separator = v
		case "sidecar_enabled":
			out.SidecarEnabled = v == "1" || v == "true"
		case "deezer_enabled":
			// Legacy key — only apply if the new key wasn't present.
			if !out.SidecarEnabled {
				out.SidecarEnabled = v == "1" || v == "true"
			}
		case "python_path":
			out.PythonPath = v
		}
	}
	return out, rows.Err()
}

// Save persists the settings, replacing existing values.
func (c *Config) Save(s Settings) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db == nil {
		return errors.New("config: nil db")
	}
	pairs := map[string]string{
		"theme_mode":               s.ThemeMode,
		"downloads_folder":         s.DownloadsFolder,
		"has_completed_onboarding": boolToOnboard(s.HasCompletedOnboarding),
		"export_lrc_file":          boolToOnboard(s.ExportLrcFile),
		"ffmpeg_path":              s.FFmpegPath,
		"audio_source":             s.AudioSource,
		"audio_format":             s.AudioFormat,
		"filename_format":          s.FilenameFormat,
		"custom_tidal_api":         s.CustomTidalAPI,
		"custom_qobuz_api":         s.CustomQobuzAPI,
		"existing_file_check_mode": s.ExistingFileCheckMode,
		"link_resolver":            s.LinkResolver,
		"auto_order":               s.AutoOrder,
		"separator":                s.Separator,
		"sidecar_enabled":          boolToOnboard(s.SidecarEnabled),
		"python_path":              s.PythonPath,
	}
	tx, err := c.db.Conn().Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for k, v := range pairs {
		if _, err := stmt.Exec(k, v); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func boolToOnboard(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// SaveFonts persists custom font definitions as JSON under the custom_fonts key.
func (c *Config) SaveFonts(fonts []map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db == nil {
		return errors.New("config: nil db")
	}
	raw, err := json.Marshal(fonts)
	if err != nil {
		return err
	}
	_, err = c.db.Conn().Exec(
		`INSERT INTO settings (key, value) VALUES ('custom_fonts', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		string(raw),
	)
	return err
}

// LoadFonts returns custom font definitions stored under the custom_fonts key.
// Returns nil if the key is missing, the database is unavailable, or the data is malformed.
func (c *Config) LoadFonts() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.db == nil {
		return nil
	}
	var raw string
	err := c.db.Conn().QueryRow(`SELECT value FROM settings WHERE key = 'custom_fonts'`).Scan(&raw)
	if err != nil {
		return nil
	}
	var fonts []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &fonts); err != nil {
		return nil
	}
	return fonts
}

// EnsureDownloadsFolder creates the downloads folder if it doesn't exist.
func EnsureDownloadsFolder(path string) error {
	if path == "" {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}

// scanSettings is unused but kept for reference when expanding the schema.
func scanSettings(rows *sql.Rows) (Settings, error) {
	var s Settings
	for rows.Next() {
		var raw json.RawMessage
		var k string
		if err := rows.Scan(&k, &raw); err != nil {
			return s, err
		}
		switch k {
		case "settings_v1":
			_ = json.Unmarshal(raw, &s)
		}
	}
	return s, rows.Err()
}

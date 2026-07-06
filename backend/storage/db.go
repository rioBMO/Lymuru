package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps an SQLite database with helpers used by the rest of the backend.
type DB struct {
	conn *sql.DB
}

// Open opens (or creates) an SQLite database at the given path. The parent
// directory is created if it does not exist. Schema migrations are applied
// via migrate().
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

// Close closes the underlying connection.
func (d *DB) Close() error {
	if d == nil || d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

// Conn returns the raw *sql.DB (for advanced use; prefer typed helpers).
func (d *DB) Conn() *sql.DB { return d.conn }

func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT UNIQUE,
			task_type TEXT NOT NULL,
			query TEXT,
			status TEXT NOT NULL,
			files TEXT,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_status ON history(status)`,
		`CREATE INDEX IF NOT EXISTS idx_history_created ON history(created_at)`,
		`CREATE TABLE IF NOT EXISTS download_history (
			id TEXT PRIMARY KEY,
			spotify_id TEXT,
			title TEXT NOT NULL,
			artists TEXT,
			album TEXT,
			duration_str TEXT,
			cover_url TEXT,
			quality TEXT,
			format TEXT,
			path TEXT,
			source TEXT,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_download_history_timestamp ON download_history(timestamp)`,
		`CREATE TABLE IF NOT EXISTS fetch_history (
			id TEXT PRIMARY KEY,
			url TEXT,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			info TEXT,
			image TEXT,
			data TEXT,
			is_explicit INTEGER DEFAULT 0,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fetch_history_timestamp ON fetch_history(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_fetch_history_type ON fetch_history(type)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := d.conn.Exec(s); err != nil {
			return fmt.Errorf("schema stmt failed: %w\n%s", err, s)
		}
	}
	return nil
}

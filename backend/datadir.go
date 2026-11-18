package backend

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// dataDirEnv overrides the data directory (useful for portable setups and tests).
const dataDirEnv = "LYMURU_DATA_DIR"

// dbFileName is the SQLite database file inside the data directory.
const dbFileName = "lymuru.db"

// ResolveDataDir returns the absolute directory for the database, logs and
// sidecar session. It is per-user (%APPDATA%\Lymuru on Windows) so the app
// works when installed to a read-only location such as Program Files.
func ResolveDataDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv(dataDirEnv)); v != "" {
		return filepath.Abs(v)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(base, "Lymuru"), nil
}

// DatabasePath returns the SQLite file path inside dataDir.
func DatabasePath(dataDir string) string {
	return filepath.Join(dataDir, dbFileName)
}

// LegacyDataDirCandidates lists the old cwd-relative "data" locations used by
// earlier builds, most specific first.
func LegacyDataDirCandidates() []string {
	var out []string
	if exe, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(exe), "data"))
	}
	if cwd, err := os.Getwd(); err == nil {
		out = append(out, filepath.Join(cwd, "data"))
	}
	return out
}

// MigrateLegacyDataDir copies an old data directory into dst when dst has no
// database yet. Only top-level regular files are copied (db, -wal, -shm,
// *.session); logs and other subdirectories are skipped. The originals are
// left untouched. It returns the source directory used, or "" if nothing was
// migrated.
func MigrateLegacyDataDir(dst string, candidates []string) (string, error) {
	if _, err := os.Stat(DatabasePath(dst)); err == nil {
		return "", nil
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return "", err
	}
	for _, src := range candidates {
		if src == "" {
			continue
		}
		srcAbs, err := filepath.Abs(src)
		if err != nil || strings.EqualFold(srcAbs, dstAbs) {
			continue
		}
		if fi, err := os.Stat(DatabasePath(srcAbs)); err != nil || !fi.Mode().IsRegular() {
			continue
		}
		if err := copyDataFiles(srcAbs, dstAbs); err != nil {
			return "", fmt.Errorf("migrate %s: %w", srcAbs, err)
		}
		return srcAbs, nil
	}
	return "", nil
}

func copyDataFiles(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	// Copy companions (WAL/SHM/session) first and the DB last, so a failure
	// never leaves a lymuru.db in dst that would stop the next attempt.
	for _, e := range entries {
		if !e.Type().IsRegular() || e.Name() == dbFileName {
			continue
		}
		if err := copyFileAtomic(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return copyFileAtomic(DatabasePath(src), DatabasePath(dst))
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".migrating"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

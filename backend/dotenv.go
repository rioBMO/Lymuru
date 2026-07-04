package backend

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv parses a simple .env file and returns the key/value pairs.
// It does not perform shell expansion or variable interpolation — values
// are taken literally after the first `=` sign. Comments start with `#`.
// If the file does not exist, an empty map and a nil error are returned
// so callers can treat "missing" as "no overrides".
func LoadDotEnv(path string) (map[string]string, error) {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes if present.
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		out[key] = val
	}
	if err := scanner.Err(); err != nil {
		return out, fmt.Errorf("read %s: %w", path, err)
	}
	return out, nil
}

// FindDotEnv searches a list of well-known locations for a .env file
// and returns the path of the first one it finds, or "" if none exist.
// Used to make the production build self-contained: the sidecar script
// is extracted from the binary to a per-user directory, and the .env is
// either found next to it or copied in from one of these locations.
func FindDotEnv() string {
	candidates := []string{
		".env",
		filepath.Join("sidecar", ".env"),
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, ".env"),
			filepath.Join(dir, "..", ".env"),
			filepath.Join(dir, "..", "..", ".env"),
		)
	}
	if cfg, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(cfg, "Lymuru", ".env"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".Lymuru", ".env"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	return ""
}

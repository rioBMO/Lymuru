package backend

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// findInstalledPython looks for python.exe in the standard python.org install
// locations on Windows and returns the newest one, or "" if none is found.
func findInstalledPython() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	var roots []string
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		roots = append(roots, filepath.Join(v, "Programs", "Python")) // per-user install
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if v := os.Getenv(env); v != "" {
			roots = append(roots, v, filepath.Join(v, "Python")) // all-users install
		}
	}
	var candidates []string
	for _, root := range roots {
		matches, _ := filepath.Glob(filepath.Join(root, "Python3*", "python.exe"))
		candidates = append(candidates, matches...)
	}
	return newestPython(candidates)
}

// newestPython picks the path whose "Python3<minor>" folder has the highest
// minor version. Folder names are compared numerically, so Python310 beats
// Python39. Paths without a parsable version are ignored.
func newestPython(paths []string) string {
	best, bestMinor := "", -1
	for _, p := range paths {
		dir := strings.ToLower(filepath.Base(filepath.Dir(p)))
		rest, ok := strings.CutPrefix(dir, "python3")
		if !ok {
			continue
		}
		// Ignore suffixes such as "-32" (32-bit) or "t" (free-threaded).
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		minor, err := strconv.Atoi(rest[:end])
		if err != nil {
			continue
		}
		if minor > bestMinor {
			best, bestMinor = p, minor
		}
	}
	return best
}

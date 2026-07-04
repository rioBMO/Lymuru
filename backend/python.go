package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// FindPythonExecutable searches for a working Python interpreter and
// returns its absolute path. It verifies each candidate by running
// `python --version` and checking the exit code.
//
// Search order (first match wins):
//  1. "python" / "python3" from PATH
//  2. Windows "py" launcher
//  3. Common install directories:
//     - %LOCALAPPDATA%\Programs\Python\Python3*\python.exe
//     - C:\Python3*\python.exe
//     - %ProgramFiles%\Python3*\python.exe
//     - %ProgramFiles(x86)%\Python3*\python.exe
func FindPythonExecutable() (string, error) {
	candidates := []string{}

	// 1. PATH lookups
	candidates = append(candidates, "python")
	if runtime.GOOS != "windows" {
		candidates = append(candidates, "python3")
	}

	// 2. Windows py launcher
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "py")
	}

	// 3. Common install directories (Windows only)
	if runtime.GOOS == "windows" {
		var searchRoots []string
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			searchRoots = append(searchRoots, filepath.Join(local, "Programs", "Python"))
		}
		searchRoots = append(searchRoots, `C:\Python314`, `C:\Python313`, `C:\Python312`, `C:\Python311`)
		if prog := os.Getenv("ProgramFiles"); prog != "" {
			searchRoots = append(searchRoots, filepath.Join(prog, "Python314"), filepath.Join(prog, "Python313"))
			// Older installs used "Program Files\Python39" etc.
			entries, _ := os.ReadDir(prog)
			for _, e := range entries {
				if !e.IsDir() || !strings.HasPrefix(strings.ToLower(e.Name()), "python") {
					continue
				}
				searchRoots = append(searchRoots, filepath.Join(prog, e.Name()))
			}
		}
		if progx86 := os.Getenv("ProgramFiles(x86)"); progx86 != "" {
			entries, _ := os.ReadDir(progx86)
			for _, e := range entries {
				if !e.IsDir() || !strings.HasPrefix(strings.ToLower(e.Name()), "python") {
					continue
				}
				searchRoots = append(searchRoots, filepath.Join(progx86, e.Name()))
			}
		}
		for _, root := range searchRoots {
			p := filepath.Join(root, "python.exe")
			candidates = append(candidates, p)
		}
	}

	return verifyCandidates(candidates)
}

// verifyCandidates tests each candidate by running `candidate --version`
// and returns the first one that exits successfully.  Returns an error
// listing all tried paths if none work.
func verifyCandidates(candidates []string) (string, error) {
	seen := map[string]bool{}
	var failures []string
	for _, c := range candidates {
		c = filepath.Clean(c)
		// If the candidate is a bare name (no separator) look it up on
		// PATH. We still do the --version check afterwards.
		if !strings.ContainsAny(c, `\/`) {
			if lp, err := exec.LookPath(c); err == nil {
				c = lp
			}
		}
		if seen[c] {
			continue
		}
		seen[c] = true
		if _, err := os.Stat(c); err != nil {
			// Not a filesystem entry — try as a bare name that LookPath already resolved.
			if !strings.ContainsAny(c, `\/`) {
				failures = append(failures, c+" (not found on PATH)")
			}
			continue
		}
		cmd := exec.Command(c, "--version")
		if out, err := cmd.Output(); err == nil {
			// Verify the output looks like a Python version.
			if strings.Contains(strings.ToLower(string(out)), "python") {
				return c, nil
			}
		}
		failures = append(failures, c)
	}
	if len(failures) == 0 {
		return "", fmt.Errorf("no Python interpreter candidates found")
	}
	return "", fmt.Errorf("Python not found. Tried: %s", strings.Join(failures, ", "))
}

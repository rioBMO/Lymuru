package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOpenTarget(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "song.lrc")
	if err := os.WriteFile(file, []byte("[00:00.00]x"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("directory returns itself", func(t *testing.T) {
		got, err := ResolveOpenTarget(dir)
		if err != nil || got != filepath.Clean(dir) {
			t.Fatalf("got %q, %v; want %q", got, err, dir)
		}
	})

	t.Run("file returns its parent", func(t *testing.T) {
		got, err := ResolveOpenTarget(file)
		if err != nil || got != filepath.Clean(dir) {
			t.Fatalf("got %q, %v; want %q", got, err, dir)
		}
	})

	for name, path := range map[string]string{
		"missing path":  filepath.Join(dir, "missing"),
		"empty string":  "",
		"relative path": "some/relative/dir",
	} {
		t.Run(name+" is an error", func(t *testing.T) {
			if got, err := ResolveOpenTarget(path); err == nil {
				t.Fatalf("got %q, want an error", got)
			}
		})
	}
}

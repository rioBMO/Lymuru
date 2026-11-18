package backend

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func withEmbeddedSidecar(t *testing.T, fsys fstest.MapFS) {
	t.Helper()
	SetEmbeddedSidecar(fsys)
	t.Cleanup(func() { SetEmbeddedSidecar(nil) })
}

func TestExtractEmbeddedSidecarWritesFiles(t *testing.T) {
	withEmbeddedSidecar(t, fstest.MapFS{
		"deezload.py":      {Data: []byte("print('hi')")},
		"requirements.txt": {Data: []byte("telethon\n")},
	})
	dataDir := t.TempDir()

	script, err := extractEmbeddedSidecar(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dataDir, "sidecar", "deezload.py"); script != want {
		t.Fatalf("script = %q, want %q", script, want)
	}
	if got := readFile(t, script); got != "print('hi')" {
		t.Errorf("script content = %q", got)
	}
	if got := readFile(t, filepath.Join(dataDir, "sidecar", "requirements.txt")); got != "telethon\n" {
		t.Errorf("requirements content = %q", got)
	}
}

func TestExtractEmbeddedSidecarSkipsUnchangedAndUpdatesChanged(t *testing.T) {
	dataDir := t.TempDir()
	withEmbeddedSidecar(t, fstest.MapFS{"deezload.py": {Data: []byte("v1")}})
	script, err := extractEmbeddedSidecar(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(script, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := extractEmbeddedSidecar(dataDir); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(script)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(old) {
		t.Errorf("unchanged script was rewritten (mtime %v, want %v)", fi.ModTime(), old)
	}

	withEmbeddedSidecar(t, fstest.MapFS{"deezload.py": {Data: []byte("v2")}})
	if _, err := extractEmbeddedSidecar(dataDir); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, script); got != "v2" {
		t.Errorf("script not updated, content = %q", got)
	}
}

func TestExtractEmbeddedSidecarErrors(t *testing.T) {
	SetEmbeddedSidecar(nil)
	if _, err := extractEmbeddedSidecar(t.TempDir()); err == nil {
		t.Error("expected error when nothing is embedded")
	}

	withEmbeddedSidecar(t, fstest.MapFS{"requirements.txt": {Data: []byte("x")}})
	if _, err := extractEmbeddedSidecar(t.TempDir()); err == nil {
		t.Error("expected error when the script is missing from the bundle")
	}
}

package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestResolveDataDirEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(dataDirEnv, dir)
	got, err := ResolveDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("got %q, want %q", got, dir)
	}
}

func TestResolveDataDirDefaultIsAbsolute(t *testing.T) {
	t.Setenv(dataDirEnv, "")
	got, err := ResolveDataDir()
	if err != nil {
		t.Skipf("no user config dir on this system: %v", err)
	}
	if !filepath.IsAbs(got) || filepath.Base(got) != "Lymuru" {
		t.Fatalf("unexpected default data dir %q", got)
	}
}

func TestMigrateLegacyDataDirCopiesFiles(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "Lymuru")
	writeFile(t, filepath.Join(src, "lymuru.db"), "db")
	writeFile(t, filepath.Join(src, "lymuru.db-wal"), "wal")
	writeFile(t, filepath.Join(src, "deezload_session.session"), "session")
	writeFile(t, filepath.Join(src, "logs", "lymuru.log"), "log")

	used, err := MigrateLegacyDataDir(dst, []string{"", filepath.Join(t.TempDir(), "missing"), src})
	if err != nil {
		t.Fatal(err)
	}
	if used != src {
		t.Fatalf("used %q, want %q", used, src)
	}
	for name, want := range map[string]string{
		"lymuru.db":                "db",
		"lymuru.db-wal":            "wal",
		"deezload_session.session": "session",
	} {
		if got := readFile(t, filepath.Join(dst, name)); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "logs")); !os.IsNotExist(err) {
		t.Errorf("logs subdirectory should not be migrated (err=%v)", err)
	}
	if got := readFile(t, filepath.Join(src, "lymuru.db")); got != "db" {
		t.Errorf("source db modified: %q", got)
	}
	if matches, _ := filepath.Glob(filepath.Join(dst, "*.migrating")); len(matches) != 0 {
		t.Errorf("leftover temp files: %v", matches)
	}
}

func TestMigrateLegacyDataDirSkipsWhenTargetHasDB(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "lymuru.db"), "old")
	writeFile(t, filepath.Join(dst, "lymuru.db"), "current")

	used, err := MigrateLegacyDataDir(dst, []string{src})
	if err != nil {
		t.Fatal(err)
	}
	if used != "" {
		t.Fatalf("expected no migration, used %q", used)
	}
	if got := readFile(t, filepath.Join(dst, "lymuru.db")); got != "current" {
		t.Fatalf("target db overwritten: %q", got)
	}
}

func TestMigrateLegacyDataDirSkipsSameDir(t *testing.T) {
	dir := t.TempDir()
	used, err := MigrateLegacyDataDir(dir, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if used != "" {
		t.Fatalf("expected no migration, used %q", used)
	}
}

func TestMigrateLegacyDataDirNoCandidates(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "Lymuru")
	used, err := MigrateLegacyDataDir(dst, []string{t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if used != "" {
		t.Fatalf("expected no migration, used %q", used)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("target dir should not be created when nothing migrates (err=%v)", err)
	}
}

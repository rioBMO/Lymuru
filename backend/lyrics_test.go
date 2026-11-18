package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// TestLiveSearchLRCLIB calls the real LRCLIB API, so it only runs on request.
func TestLiveSearchLRCLIB(t *testing.T) {
	if os.Getenv("LYMURU_LIVE_TESTS") == "" {
		t.Skip("set LYMURU_LIVE_TESTS=1 to run tests against live services")
	}
	lyrics, synced, err := SearchLRCLIB("Rick Astley", "Never Gonna Give You Up")
	if err != nil {
		t.Fatalf("SearchLRCLIB failed: %v", err)
	}

	if lyrics == "" {
		t.Error("Expected lyrics, got empty string")
	}

	if !synced {
		t.Log("Note: expected synced lyrics for Rick Astley, but got unsynced (this is fine depending on LRCLIB's database)")
	}
}

func TestSimplifyTrackName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Song Title (feat. Artist)", "Song Title"},
		{"Song Title - Remastered 2020", "Song Title"},
		{"Song Title (Radio Edit)", "Song Title"},
		{"Normal Song", "Normal Song"},
	}

	for _, c := range cases {
		output := simplifyTrackName(c.input)
		if output != c.expected {
			t.Errorf("simplifyTrackName(%q) == %q, expected %q", c.input, output, c.expected)
		}
	}
}

// fakeLRCLIB serves /api/get and /api/search from fixed data keyed by track
// name, and counts requests.
func fakeLRCLIB(t *testing.T) (*LyricsClient, *atomic.Int32) {
	t.Helper()
	get := map[string]LRCLibResponse{
		"Synced Song": {TrackName: "Synced Song", SyncedLyrics: "[00:01.50]First line\n[01:02.03]Second line"},
		"Plain Song":  {TrackName: "Plain Song", PlainLyrics: "First line\nSecond line"},
	}
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		track := r.URL.Query().Get("track_name")
		switch r.URL.Path {
		case "/api/get":
			res, ok := get[track]
			if !ok {
				http.NotFound(w, r)
				return
			}
			json.NewEncoder(w).Encode(res)
		case "/api/search":
			w.Write([]byte("[]"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := NewLyricsClient()
	c.baseURL = srv.URL
	return c, &hits
}

func lyricsRequest(track, dir string) LyricsDownloadRequest {
	return LyricsDownloadRequest{
		TrackName:      track,
		ArtistName:     "Artist",
		AlbumName:      "Album",
		OutputDir:      dir,
		FilenameFormat: "{artist} - {title}",
		DurationMs:     180000,
	}
}

func TestDownloadLyricsWritesSyncedLRC(t *testing.T) {
	c, _ := fakeLRCLIB(t)
	dir := t.TempDir()

	resp, err := c.DownloadLyrics(lyricsRequest("Synced Song", dir))
	if err != nil || !resp.Success {
		t.Fatalf("DownloadLyrics failed: resp=%+v err=%v", resp, err)
	}
	want := filepath.Join(dir, "Artist - Synced Song.lrc")
	if resp.File != want {
		t.Errorf("File = %q, want %q", resp.File, want)
	}
	got, _ := os.ReadFile(want)
	for _, line := range []string{"[00:01.50]First line", "[01:02.03]Second line"} {
		if !strings.Contains(string(got), line) {
			t.Errorf("file is missing %q:\n%s", line, got)
		}
	}
}

func TestDownloadLyricsWritesPlainLRCWithoutTimestamps(t *testing.T) {
	c, _ := fakeLRCLIB(t)
	dir := t.TempDir()

	resp, err := c.DownloadLyrics(lyricsRequest("Plain Song", dir))
	if err != nil || !resp.Success {
		t.Fatalf("DownloadLyrics failed: resp=%+v err=%v", resp, err)
	}
	got, _ := os.ReadFile(resp.File)
	if !strings.Contains(string(got), "\nFirst line\nSecond line\n") {
		t.Errorf("plain lines missing:\n%s", got)
	}
	if strings.Contains(string(got), "[00:") {
		t.Errorf("plain lyrics got timestamps:\n%s", got)
	}
}

func TestDownloadLyricsReportsNotFoundWithoutWritingAFile(t *testing.T) {
	c, _ := fakeLRCLIB(t)
	dir := t.TempDir()

	resp, err := c.DownloadLyrics(lyricsRequest("Unknown Song", dir))
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	if resp.Success || !resp.NotFound || resp.Error == "" {
		t.Errorf("want NotFound failure, got %+v", resp)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Artist - Unknown Song.lrc")); !os.IsNotExist(statErr) {
		t.Error("a file was written although no lyrics were found")
	}
}

func TestDownloadLyricsSkipsExistingFile(t *testing.T) {
	c, hits := fakeLRCLIB(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Artist - Synced Song.lrc"), []byte("old"), 0o644)

	resp, err := c.DownloadLyrics(lyricsRequest("Synced Song", dir))
	if err != nil || !resp.Success || !resp.AlreadyExists {
		t.Fatalf("want AlreadyExists, got resp=%+v err=%v", resp, err)
	}
	if hits.Load() != 0 {
		t.Errorf("server was called %d times, want 0", hits.Load())
	}
}

func TestLyricsQueryForUsesOverridesWithoutReadingTheFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.flac")

	artist, title, err := LyricsQueryFor(missing, " Artist ", "Title")
	if err != nil || artist != "Artist" || title != "Title" {
		t.Errorf("got (%q, %q, %v), want (Artist, Title, nil)", artist, title, err)
	}
}

func TestLyricsQueryForFailsWhenTagsCantFillTheGaps(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.flac")

	for _, tc := range [][2]string{{"", ""}, {"Artist", ""}, {"", "Title"}} {
		if _, _, err := LyricsQueryFor(missing, tc[0], tc[1]); err == nil {
			t.Errorf("LyricsQueryFor(missing, %q, %q): want an error", tc[0], tc[1])
		}
	}
}

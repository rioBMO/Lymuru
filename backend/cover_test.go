package backend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// imageServer serves fixed bytes at /ok and 404 elsewhere, counting requests.
func imageServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/ok" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("jpeg-bytes"))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func coverRequest(url, dir string) CoverDownloadRequest {
	return CoverDownloadRequest{
		CoverURL:       url,
		TrackName:      "Song",
		ArtistName:     "Artist",
		OutputDir:      dir,
		FilenameFormat: "{artist} - {title}",
	}
}

func TestDownloadCoverWritesFileNamedLikeTheTrack(t *testing.T) {
	srv, _ := imageServer(t)
	dir := t.TempDir()

	resp, err := NewCoverClient().DownloadCover(coverRequest(srv.URL+"/ok", dir))
	if err != nil || !resp.Success {
		t.Fatalf("DownloadCover failed: resp=%+v err=%v", resp, err)
	}
	want := filepath.Join(dir, "Artist - Song.jpg")
	if resp.File != want {
		t.Errorf("File = %q, want %q", resp.File, want)
	}
	got, _ := os.ReadFile(want)
	if string(got) != "jpeg-bytes" {
		t.Errorf("file content = %q", got)
	}
}

func TestDownloadCoverSkipsExistingFile(t *testing.T) {
	srv, hits := imageServer(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Artist - Song.jpg"), []byte("old"), 0o644)

	resp, err := NewCoverClient().DownloadCover(coverRequest(srv.URL+"/ok", dir))
	if err != nil || !resp.Success || !resp.AlreadyExists {
		t.Fatalf("want AlreadyExists, got resp=%+v err=%v", resp, err)
	}
	if hits.Load() != 0 {
		t.Errorf("server was called %d times, want 0", hits.Load())
	}
}

func TestDownloadCoverReportsHTTPError(t *testing.T) {
	srv, _ := imageServer(t)
	dir := t.TempDir()

	resp, err := NewCoverClient().DownloadCover(coverRequest(srv.URL+"/missing", dir))
	if err == nil || resp == nil || resp.Success || resp.Error == "" {
		t.Fatalf("want a failure response, got resp=%+v err=%v", resp, err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Artist - Song.jpg")); !os.IsNotExist(statErr) {
		t.Error("a file was written for a failed download")
	}
}

func TestDownloadCoverRequiresURL(t *testing.T) {
	resp, err := NewCoverClient().DownloadCover(coverRequest("", t.TempDir()))
	if err == nil || resp == nil || resp.Success {
		t.Fatalf("want a failure response, got resp=%+v err=%v", resp, err)
	}
}

func TestDownloadArtistImages(t *testing.T) {
	srv, _ := imageServer(t)
	dir := t.TempDir()
	c := NewCoverClient()
	url := srv.URL + "/ok"

	header, err := c.DownloadHeader(HeaderDownloadRequest{HeaderURL: url, ArtistName: "Band", OutputDir: dir})
	if err != nil || !header.Success {
		t.Fatalf("DownloadHeader: resp=%+v err=%v", header, err)
	}
	avatar, err := c.DownloadAvatar(AvatarDownloadRequest{AvatarURL: url, ArtistName: "Band", OutputDir: dir})
	if err != nil || !avatar.Success {
		t.Fatalf("DownloadAvatar: resp=%+v err=%v", avatar, err)
	}
	gallery, err := c.DownloadGalleryImage(GalleryImageDownloadRequest{ImageURL: url, ArtistName: "Band", ImageIndex: 0, OutputDir: dir})
	if err != nil || !gallery.Success {
		t.Fatalf("DownloadGalleryImage: resp=%+v err=%v", gallery, err)
	}

	for _, name := range []string{"Band_Header.jpg", "Band_Avatar.jpg", "Band_Gallery_1.jpg"} {
		if _, err := os.Stat(filepath.Join(dir, "Band", name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}

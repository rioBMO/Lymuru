package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LyricsResult is the shape returned by LRCLIB.
type LyricsResult struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

// LRCLIBBase is the public LRCLIB endpoint. Override for testing.
var LRCLIBBase = "https://lrclib.net/api"

// SearchLyrics queries LRCLIB for lyrics matching the given artist and title.
// Returns nil, nil if no match.
func SearchLyrics(ctx context.Context, artist, title string) (*LyricsResult, error) {
	a := strings.TrimSpace(artist)
	t := strings.TrimSpace(title)
	if a == "" || t == "" {
		return nil, fmt.Errorf("artist and title required")
	}
	u := fmt.Sprintf("%s/get?artist_name=%s&track_name=%s",
		LRCLIBBase, url.QueryEscape(a), url.QueryEscape(t))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib: HTTP %d", resp.StatusCode)
	}
	var out LyricsResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

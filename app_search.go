package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lymuru/lymuru/backend"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type SpotifyMetadataRequest struct {
	URL       string  `json:"url"`
	Batch     bool    `json:"batch"`
	Delay     float64 `json:"delay"`
	Timeout   float64 `json:"timeout"`
	Separator string  `json:"separator,omitempty"`
}

type SpotifySearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type SpotifySearchByTypeRequest struct {
	Query      string `json:"query"`
	SearchType string `json:"search_type"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

type CheckFileExistenceRequest struct {
	SpotifyID           string `json:"spotify_id"`
	TrackName           string `json:"track_name"`
	ArtistName          string `json:"artist_name"`
	Artists             string `json:"artists,omitempty"`
	AlbumName           string `json:"album_name,omitempty"`
	AlbumArtist         string `json:"album_artist,omitempty"`
	Category            string `json:"category,omitempty"`
	UPC                 string `json:"upc,omitempty"`
	ReleaseDate         string `json:"release_date,omitempty"`
	ISRC                string `json:"isrc,omitempty"`
	TrackNumber         int    `json:"track_number,omitempty"`
	DiscNumber          int    `json:"disc_number,omitempty"`
	TotalTracks         int    `json:"total_tracks,omitempty"`
	TotalDiscs          int    `json:"total_discs,omitempty"`
	Position            int    `json:"position,omitempty"`
	UseAlbumTrackNumber bool   `json:"use_album_track_number,omitempty"`
	FilenameFormat      string `json:"filename_format,omitempty"`
	IncludeTrackNumber  bool   `json:"include_track_number,omitempty"`
	AudioFormat         string `json:"audio_format,omitempty"`
	RelativePath        string `json:"relative_path,omitempty"`
}

type CheckFileExistenceResult struct {
	SpotifyID  string `json:"spotify_id"`
	Exists     bool   `json:"exists"`
	FilePath   string `json:"file_path,omitempty"`
	TrackName  string `json:"track_name,omitempty"`
	ArtistName string `json:"artist_name,omitempty"`
}

func (a *App) GetSpotifyMetadata(req SpotifyMetadataRequest) (string, error) {
	if req.URL == "" {
		return "", fmt.Errorf("URL parameter is required")
	}

	if req.Delay == 0 {
		req.Delay = 1.0
	}
	if req.Timeout == 0 {
		req.Timeout = 300.0
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout*float64(time.Second)))
	defer cancel()

	separator := req.Separator
	if separator == "" {
		separator = ", "
	}

	data, err := backend.GetFilteredSpotifyData(ctx, req.URL, req.Batch, time.Duration(req.Delay*float64(time.Second)), separator, func(tracks interface{}) {
		runtime.EventsEmit(a.ctx, "metadata-stream", tracks)
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch metadata: %v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %v", err)
	}

	// Record fetch in history.
	if a.history != nil {
		historyType := parseSpotifyURLType(req.URL)
		_ = a.history.AddFetchHistoryItem(backend.FetchHistoryItem{
			ID:        fmt.Sprintf("%s-%d", req.URL, time.Now().Unix()),
			URL:       req.URL,
			Type:      historyType,
			Name:      req.URL,
			Timestamp: time.Now().Unix(),
		})
	}

	return string(jsonData), nil
}

// parseSpotifyURLType extracts the entity type from a Spotify URL.
func parseSpotifyURLType(rawURL string) string {
	u := strings.ToLower(rawURL)
	if strings.Contains(u, "/track/") {
		return "track"
	}
	if strings.Contains(u, "/album/") {
		return "album"
	}
	if strings.Contains(u, "/playlist/") {
		return "playlist"
	}
	if strings.Contains(u, "/artist/") {
		return "artist"
	}
	return "unknown"
}

func (a *App) SearchSpotify(req SpotifySearchRequest) (*backend.SearchResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return backend.SearchSpotify(ctx, req.Query, req.Limit)
}

func (a *App) SearchSpotifyByType(req SpotifySearchByTypeRequest) ([]backend.SearchResult, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	if req.SearchType == "" {
		return nil, fmt.Errorf("search type is required")
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return backend.SearchSpotifyByType(ctx, req.Query, req.SearchType, req.Limit, req.Offset)
}

func (a *App) GetTrackISRC(spotifyTrackID string) string {
	return backend.ResolveTrackISRC(spotifyTrackID)
}

func (a *App) GetPreviewURL(trackID string) (string, error) {
	return backend.GetPreviewURL(trackID)
}

func (a *App) SkipDownloadItem(itemID, filePath string) {
	backend.SkipDownloadItem(itemID, filePath)
}

func (a *App) GetRecentFetches() (string, error) {
	data, err := backend.LoadRecentFetches()
	if err != nil {
		return "[]", err
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "[]", err
	}
	return string(jsonData), nil
}

func (a *App) SaveRecentFetches(data string) error {
	var fetches []backend.RecentFetchItem
	if err := json.Unmarshal([]byte(data), &fetches); err != nil {
		return err
	}
	return backend.SaveRecentFetches(fetches)
}

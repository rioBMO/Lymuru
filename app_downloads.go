package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lymuru/lymuru/backend"
)

type DownloadRequest struct {
	Service              string `json:"service"`
	Query                string `json:"query,omitempty"`
	TrackName            string `json:"track_name,omitempty"`
	ArtistName           string `json:"artist_name,omitempty"`
	AlbumName            string `json:"album_name,omitempty"`
	AlbumArtist          string `json:"album_artist,omitempty"`
	ReleaseDate          string `json:"release_date,omitempty"`
	CoverURL             string `json:"cover_url,omitempty"`
	TidalAPIURL          string `json:"tidal_api_url,omitempty"`
	QobuzAPIURL          string `json:"qobuz_api_url,omitempty"`
	OutputDir            string `json:"output_dir,omitempty"`
	AudioFormat          string `json:"audio_format,omitempty"`
	FilenameFormat       string `json:"filename_format,omitempty"`
	TrackNumber          bool   `json:"track_number,omitempty"`
	Position             int    `json:"position,omitempty"`
	UseAlbumTrackNumber  bool   `json:"use_album_track_number,omitempty"`
	SpotifyID            string `json:"spotify_id,omitempty"`
	EmbedLyrics          bool   `json:"embed_lyrics,omitempty"`
	EmbedMaxQualityCover bool   `json:"embed_max_quality_cover,omitempty"`
	ServiceURL           string `json:"service_url,omitempty"`
	Duration             int    `json:"duration,omitempty"`
	ItemID               string `json:"item_id,omitempty"`
	SpotifyTrackNumber   int    `json:"spotify_track_number,omitempty"`
	SpotifyDiscNumber    int    `json:"spotify_disc_number,omitempty"`
	SpotifyTotalTracks   int    `json:"spotify_total_tracks,omitempty"`
	SpotifyTotalDiscs    int    `json:"spotify_total_discs,omitempty"`
	ISRC                 string `json:"isrc,omitempty"`
	Copyright            string `json:"copyright,omitempty"`
	Publisher            string `json:"publisher,omitempty"`
	Composer             string `json:"composer,omitempty"`
	PlaylistName         string `json:"playlist_name,omitempty"`
	PlaylistOwner        string `json:"playlist_owner,omitempty"`
	AllowFallback        bool   `json:"allow_fallback"`
	UseFirstArtistOnly   bool   `json:"use_first_artist_only,omitempty"`
	UseSingleGenre       bool   `json:"use_single_genre,omitempty"`
	EmbedGenre           bool   `json:"embed_genre,omitempty"`
	Separator            string `json:"separator,omitempty"`
	SaveCover            bool   `json:"save_cover,omitempty"`
	Artists              string `json:"artists,omitempty"`
	Category             string `json:"category,omitempty"`
	UPC                  string `json:"upc,omitempty"`
}

type DownloadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	File          string `json:"file,omitempty"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
	Cancelled     bool   `json:"cancelled,omitempty"`
	ItemID        string `json:"item_id,omitempty"`
	SourceURL     string `json:"source_url,omitempty"`
	SourceLabel   string `json:"source_label,omitempty"`
}

// DownloadTrack routes a download request to the appropriate provider
// (Tidal, Amazon, or Qobuz) and returns the result.
func (a *App) DownloadTrack(req DownloadRequest) (DownloadResponse, error) {
	if req.OutputDir == "" {
		req.OutputDir = "."
	}
	req.OutputDir = filepath.Clean(req.OutputDir)

	if err := os.MkdirAll(req.OutputDir, 0o755); err != nil {
		return DownloadResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create output directory: %v", err),
			ItemID:  req.ItemID,
		}, err
	}

	// Notify the progress system.
	backend.SetDownloading(true)
	backend.StartDownloadItem(req.ItemID)
	defer backend.SetDownloading(false)

	// Defaults for optional fields.
	if req.Separator == "" {
		req.Separator = ", "
	}
	if req.FilenameFormat == "" {
		req.FilenameFormat = "title-artist"
	}

	sep := req.Separator

	var filename string
	var sourceURL string
	var sourceLabel string
	var err error

	switch strings.ToLower(req.Service) {
	case "tidal":
		if req.SpotifyID == "" {
			return DownloadResponse{Success: false, Error: "spotify_id is required for Tidal", ItemID: req.ItemID}, fmt.Errorf("spotify_id required")
		}

		tidalAPI := req.TidalAPIURL
		if tidalAPI == "" {
			tidalAPI = getDefaultTidalAPI()
		}
		td := backend.NewTidalDownloader(tidalAPI)
		filename, err = td.Download(
			req.SpotifyID, req.OutputDir, req.AudioFormat, req.FilenameFormat,
			req.TrackNumber, req.Position,
			req.TrackName, req.ArtistName, req.AlbumName, req.AlbumArtist, req.ReleaseDate,
			req.UseAlbumTrackNumber, req.CoverURL, req.EmbedMaxQualityCover,
			req.SpotifyTrackNumber, req.SpotifyDiscNumber, req.SpotifyTotalTracks, req.SpotifyTotalDiscs,
			req.Copyright, req.Publisher, req.Composer,
			sep, req.ISRC, "https://open.spotify.com/track/"+req.SpotifyID,
			req.AllowFallback, req.UseFirstArtistOnly, req.UseSingleGenre, req.EmbedGenre,
		)
		sourceURL = td.SourceURL
		sourceLabel = "Tidal"

	case "amazon":
		if req.SpotifyID == "" {
			return DownloadResponse{Success: false, Error: "spotify_id is required for Amazon", ItemID: req.ItemID}, fmt.Errorf("spotify_id required")
		}

		ad := backend.NewAmazonDownloader()
		filename, err = ad.DownloadBySpotifyID(
			req.SpotifyID, req.OutputDir, req.AudioFormat, req.FilenameFormat,
			req.PlaylistName, req.PlaylistOwner,
			req.TrackNumber, req.Position,
			req.TrackName, req.ArtistName, req.AlbumName, req.AlbumArtist, req.ReleaseDate,
			req.CoverURL,
			req.SpotifyTrackNumber, req.SpotifyDiscNumber, req.SpotifyTotalTracks,
			req.EmbedMaxQualityCover,
			req.SpotifyTotalDiscs,
			req.Copyright, req.Publisher, req.Composer,
			sep, req.ISRC, "https://open.spotify.com/track/"+req.SpotifyID,
			req.UseFirstArtistOnly, req.UseSingleGenre, req.EmbedGenre,
		)
		sourceURL = ad.SourceURL
		sourceLabel = "Amazon"

	case "qobuz":
		if req.SpotifyID == "" {
			return DownloadResponse{Success: false, Error: "spotify_id is required for Qobuz", ItemID: req.ItemID}, fmt.Errorf("spotify_id required")
		}

		qd := backend.NewQobuzDownloader()
		if req.QobuzAPIURL != "" {
			qd.SetCustomAPIURL(req.QobuzAPIURL)
		}
		filename, err = qd.DownloadTrack(
			req.SpotifyID, req.OutputDir, req.AudioFormat, req.FilenameFormat,
			req.TrackNumber, req.Position,
			req.TrackName, req.ArtistName, req.AlbumName, req.AlbumArtist, req.ReleaseDate,
			req.UseAlbumTrackNumber, req.CoverURL, req.EmbedMaxQualityCover,
			req.SpotifyTrackNumber, req.SpotifyDiscNumber,
			req.SpotifyTotalTracks, req.SpotifyTotalDiscs,
			req.Copyright, req.Publisher, req.Composer,
			sep, "https://open.spotify.com/track/"+req.SpotifyID,
			req.AllowFallback, req.UseFirstArtistOnly, req.UseSingleGenre, req.EmbedGenre,
		)
		sourceURL = qd.SourceURL
		sourceLabel = qd.SourceLabel
		if sourceLabel == "" {
			sourceLabel = "Qobuz"
		}

	default:
		return DownloadResponse{
			Success: false,
			Error:   fmt.Sprintf("unsupported service: %s", req.Service),
			ItemID:  req.ItemID,
		}, fmt.Errorf("unsupported service: %s", req.Service)
	}

	if err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(filename, "EXISTS:") {
			return DownloadResponse{
				Success:       true,
				Message:       "File already exists",
				File:          strings.TrimPrefix(filename, "EXISTS:"),
				AlreadyExists: true,
				ItemID:        req.ItemID,
				SourceURL:     sourceURL,
				SourceLabel:   sourceLabel,
			}, nil
		}
		return DownloadResponse{
			Success:     false,
			Error:       errMsg,
			ItemID:      req.ItemID,
			SourceURL:   sourceURL,
			SourceLabel: sourceLabel,
		}, err
	}

	return DownloadResponse{
		Success:     true,
		Message:     "Downloaded successfully",
		File:        filename,
		ItemID:      req.ItemID,
		SourceURL:   sourceURL,
		SourceLabel: sourceLabel,
	}, nil
}

// getDefaultTidalAPI returns the default Tidal community API URL.
func getDefaultTidalAPI() string {
	custom := backend.GetCustomTidalAPISetting()
	if custom != "" {
		return strings.TrimRight(strings.TrimSpace(custom), "/")
	}
	return "https://tidal.spotbye.qzz.io"
}

// GetStreamingURLs resolves a Spotify track ID to Tidal and Amazon streaming
// URLs via SongLink, returning the result as JSON.
func (a *App) GetStreamingURLs(spotifyTrackID string, region string) (string, error) {
	client := backend.NewSongLinkClient()
	urls, err := client.GetAllURLsFromSpotify(spotifyTrackID, region)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(urls)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) GetDownloadProgress() backend.ProgressInfo {
	return backend.GetDownloadProgress()
}

func (a *App) GetDownloadQueue() backend.DownloadQueueInfo {
	return backend.GetDownloadQueue()
}

func (a *App) ClearCompletedDownloads() {
	// Clear only completed items from the download queue.
	backend.ClearDownloadQueue()
}

func (a *App) ClearAllDownloads() {
	backend.ClearAllDownloads()
}

func (a *App) AddToDownloadQueue(spotifyID, trackName, artistName, albumName string) string {
	backend.AddToQueue("", trackName, artistName, albumName, spotifyID)
	return spotifyID
}

func (a *App) MarkDownloadItemFailed(itemID, errorMsg string) {
	// stub — queue is managed in-memory; failing items is handled
	// by the provider error paths
}

func (a *App) ForceStopDownloads() {
	// stub — stop is managed by cancellation scopes inside providers
}

func (a *App) ExportFailedDownloads() (string, error) {
	return "", nil // Stub
}

// ---------------------------------------------------------------------------
// File existence / utility stubs (used by SpotiFLAC frontend)
// ---------------------------------------------------------------------------

// CheckFilesExistence checks if files matching the given track metadata
// already exist on disk. Stub: always returns empty (no existing files).
func (a *App) CheckFilesExistence(outputDir, rootDir string, tracks []CheckFileExistenceRequest) []CheckFileExistenceResult {
	results := make([]CheckFileExistenceResult, 0, len(tracks))
	for _, t := range tracks {
		results = append(results, CheckFileExistenceResult{
			SpotifyID:  t.SpotifyID,
			Exists:     false,
			TrackName:  t.TrackName,
			ArtistName: t.ArtistName,
		})
	}
	return results
}

// CreateM3U8File writes an .m3u8 playlist file (stub).
func (a *App) CreateM3U8File(m3u8Name, outputDir string, filePaths []string) error {
	return nil // stub
}

// CreateLogFile writes a download log file (stub).
func (a *App) CreateLogFile(fileName, outputDir string, logs []string) error {
	return nil // stub
}

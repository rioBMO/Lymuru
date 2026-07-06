package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lymuru/lymuru/backend"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
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
	defer func() {
		backend.SetDownloading(false)
	}()

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
			backend.FailDownloadItem(req.ItemID, "spotify_id is required for Tidal")
			return DownloadResponse{Success: false, Error: "spotify_id is required for Tidal", ItemID: req.ItemID}, fmt.Errorf("spotify_id required")
		}

		tidalAPI := req.TidalAPIURL
		if tidalAPI == "" {
			tidalAPI = getDefaultTidalAPI()
		}
		fmt.Printf("Using Tidal API: %s\n", tidalAPI)
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
			backend.FailDownloadItem(req.ItemID, "spotify_id is required for Amazon")
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
			backend.FailDownloadItem(req.ItemID, "spotify_id is required for Qobuz")
			return DownloadResponse{Success: false, Error: "spotify_id is required for Qobuz", ItemID: req.ItemID}, fmt.Errorf("spotify_id required")
		}

		qd := backend.NewQobuzDownloader()
		qobuzAPI := req.QobuzAPIURL
		if qobuzAPI == "" {
			qobuzAPI = backend.GetCustomQobuzAPISetting()
		}
		if qobuzAPI != "" {
			fmt.Printf("Using custom Qobuz API: %s\n", qobuzAPI)
			qd.SetCustomAPIURL(qobuzAPI)
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

	case "deezer":
		if a.sidecar == nil || !a.sidecar.IsRunning() {
			backend.FailDownloadItem(req.ItemID, "Deezer sidecar is not running")
			return DownloadResponse{
				Success: false,
				Error:   "Deezer sidecar is not running",
				ItemID:  req.ItemID,
			}, fmt.Errorf("sidecar not running")
		}
		if req.TrackName == "" || req.ArtistName == "" {
			backend.FailDownloadItem(req.ItemID, "track name and artist are required for Deezer")
			return DownloadResponse{
				Success: false,
				Error:   "track name and artist are required for Deezer",
				ItemID:  req.ItemID,
			}, fmt.Errorf("track info required")
		}
		search, searchErr := a.sidecar.Search(req.ArtistName, req.TrackName)
		if searchErr != nil {
			backend.FailDownloadItem(req.ItemID, fmt.Sprintf("Deezer search: %v", searchErr))
			return DownloadResponse{
				Success: false,
				Error:   fmt.Sprintf("Deezer search: %v", searchErr),
				ItemID:  req.ItemID,
			}, searchErr
		}
		searchKey, _ := search["search_key"].(string)
		if searchKey == "" {
			backend.FailDownloadItem(req.ItemID, "No results found on Deezer")
			return DownloadResponse{
				Success: false,
				Error:   "No results found on Deezer",
				ItemID:  req.ItemID,
			}, fmt.Errorf("no deezer results")
		}
		// Read LRC export setting for the sidecar.
		settings, _ := a.config.Load()
		exportLrc := settings.ExportLrcFile
		filePath, dlErr := a.sidecar.Download(searchKey, 0, req.OutputDir, exportLrc)
		if dlErr != nil {
			backend.FailDownloadItem(req.ItemID, fmt.Sprintf("Deezer download: %v", dlErr))
			return DownloadResponse{
				Success: false,
				Error:   fmt.Sprintf("Deezer download: %v", dlErr),
				ItemID:  req.ItemID,
			}, dlErr
		}
		filename = filePath
		sourceURL = "https://deezer.com"
		sourceLabel = "Deezer"

	default:
		errMsg := fmt.Sprintf("unsupported service: %s", req.Service)
		backend.FailDownloadItem(req.ItemID, errMsg)
		return DownloadResponse{
			Success: false,
			Error:   errMsg,
			ItemID:  req.ItemID,
		}, fmt.Errorf("%s", errMsg)
	}

	if err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(filename, "EXISTS:") {
			existingFile := strings.TrimPrefix(filename, "EXISTS:")
			backend.SkipDownloadItem(req.ItemID, existingFile)
			return DownloadResponse{
				Success:       true,
				Message:       "File already exists",
				File:          existingFile,
				AlreadyExists: true,
				ItemID:        req.ItemID,
				SourceURL:     sourceURL,
				SourceLabel:   sourceLabel,
			}, nil
		}
		backend.FailDownloadItem(req.ItemID, errMsg)
		return DownloadResponse{
			Success:     false,
			Error:       errMsg,
			ItemID:      req.ItemID,
			SourceURL:   sourceURL,
			SourceLabel: sourceLabel,
		}, err
	}

	// Record successful download in history.
	if a.history != nil {
		now := time.Now().Unix()
		_ = a.history.AddDownloadHistoryItem(backend.DownloadHistoryItem{
			ID:        fmt.Sprintf("%s-%d", req.SpotifyID, now),
			SpotifyID: req.SpotifyID,
			Title:     req.TrackName,
			Artists:   req.ArtistName,
			Album:     req.AlbumName,
			CoverURL:  req.CoverURL,
			Format:    req.AudioFormat,
			Path:      filename,
			Source:    sourceLabel,
			Timestamp: now,
		})
	}

	// Mark the queue item as completed with the final file size.
	var finalSizeMB float64
	if fi, statErr := os.Stat(filename); statErr == nil {
		finalSizeMB = float64(fi.Size()) / (1024 * 1024)
	}
	backend.CompleteDownloadItem(req.ItemID, filename, finalSizeMB)

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
	backend.FailDownloadItem(itemID, errorMsg)
}

func (a *App) MarkDownloadItemCompleted(itemID, filePath string, finalSizeMB float64) {
	backend.CompleteDownloadItem(itemID, filePath, finalSizeMB)
}

func (a *App) MarkDownloadItemSkipped(itemID, filePath string) {
	backend.SkipDownloadItem(itemID, filePath)
}

func (a *App) ForceStopDownloads() {
	backend.ForceStopActiveDownloads()
}

func (a *App) ExportFailedDownloads() (string, error) {
	queue := backend.GetDownloadQueue().Queue
	var failed []backend.DownloadItem
	for _, item := range queue {
		if item.Status == backend.StatusFailed {
			failed = append(failed, item)
		}
	}
	if len(failed) == 0 {
		return "No failed downloads to export", nil
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	defaultName := "failed_downloads_" + timestamp + ".txt"

	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Text files (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return "Export cancelled", nil
	}

	var lines []string
	for _, item := range failed {
		lines = append(lines, fmt.Sprintf("%s - %s | %s", item.TrackName, item.ArtistName, item.ErrorMessage))
	}
	if writeErr := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); writeErr != nil {
		return "", fmt.Errorf("failed to write export file: %v", writeErr)
	}

	return fmt.Sprintf("Successfully exported %d failed downloads", len(failed)), nil
}

// ---------------------------------------------------------------------------
// File existence / utility
// ---------------------------------------------------------------------------

// CheckFilesExistence checks if files matching the given track metadata
// already exist on disk by constructing the expected filename and checking
// with os.Stat. Supports filename-format tokens and audio format extensions.
func (a *App) CheckFilesExistence(outputDir, rootDir string, tracks []CheckFileExistenceRequest) []CheckFileExistenceResult {
	if len(tracks) == 0 {
		return nil
	}

	outputDir = backend.NormalizePath(outputDir)

	const defaultFilenameFormat = "title-artist"

	results := make([]CheckFileExistenceResult, 0, len(tracks))
	for _, t := range tracks {
		res := CheckFileExistenceResult{
			SpotifyID:  t.SpotifyID,
			TrackName:  t.TrackName,
			ArtistName: t.ArtistName,
			Exists:     false,
		}

		if strings.TrimSpace(t.TrackName) == "" || strings.TrimSpace(t.ArtistName) == "" {
			results = append(results, res)
			continue
		}

		// Resolve filename format with tokens.
		filenameFormat := t.FilenameFormat
		if filenameFormat == "" {
			filenameFormat = defaultFilenameFormat
		}
		if strings.Contains(filenameFormat, "{") {
			artistsForTokens := t.Artists
			if strings.TrimSpace(artistsForTokens) == "" {
				artistsForTokens = t.ArtistName
			}
			filenameFormat = backend.ApplyExtraFilenameTokens(filenameFormat, artistsForTokens, t.TotalTracks, t.TotalDiscs)
			filenameFormat = backend.ApplyFilenameContextTokens(filenameFormat, t.Category, "", "", t.UPC)
		}

		// Determine the file extension.
		fileExt := ".flac"
		switch strings.ToLower(strings.TrimSpace(t.AudioFormat)) {
		case "mp3":
			fileExt = ".mp3"
		case "m4a", "m4a-aac", "m4a-alac", "alac", "atmos", "apple":
			fileExt = ".m4a"
		}

		trackNumber := t.Position
		if t.UseAlbumTrackNumber && t.TrackNumber > 0 {
			trackNumber = t.TrackNumber
		}

		expectedFilename := backend.BuildExpectedFilename(
			t.TrackName,
			t.ArtistName,
			t.AlbumName,
			t.AlbumArtist,
			t.ReleaseDate,
			filenameFormat,
			"", "",
			t.IncludeTrackNumber,
			trackNumber,
			t.DiscNumber,
			t.UseAlbumTrackNumber,
		)
		expectedFilename = strings.TrimSuffix(expectedFilename, ".flac") + fileExt

		targetDir := outputDir
		if strings.TrimSpace(t.RelativePath) != "" {
			targetDir = filepath.Join(outputDir, t.RelativePath)
		}

		expectedPath := filepath.Join(targetDir, expectedFilename)

		if info, err := os.Stat(expectedPath); err == nil && info.Size() > 100*1024 {
			res.Exists = true
			res.FilePath = expectedPath
		}

		results = append(results, res)
	}

	return results
}

// CreateM3U8File writes an .m3u8 playlist file with paths relative to outputDir.
func (a *App) CreateM3U8File(m3u8Name, outputDir string, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	safeName := backend.SanitizeFilename(m3u8Name)
	if safeName == "" {
		safeName = "playlist"
	}

	m3u8Path := filepath.Join(outputDir, safeName+".m3u8")

	f, err := os.Create(m3u8Path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		return err
	}

	for _, path := range filePaths {
		if path == "" {
			continue
		}
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)
		if _, err := f.WriteString(relPath + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// CreateLogFile writes a download log file.
func (a *App) CreateLogFile(fileName, outputDir string, logs []string) error {
	if len(logs) == 0 {
		return nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(outputDir, fileName+".txt")
	return os.WriteFile(path, []byte(strings.Join(logs, "\n")), 0644)
}

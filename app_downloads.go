package main

import (
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

// DownloadTrack executes a native Go download.
func (a *App) DownloadTrack(req DownloadRequest) (DownloadResponse, error) {
	// Stub for now, full integration happens in phase 3/4.
	return DownloadResponse{
		Success: true,
		Message: "Stubbed download track",
		ItemID:  req.ItemID,
	}, nil
}

func (a *App) GetStreamingURLs(spotifyTrackID string, region string) (string, error) {
	return "", nil // Stub
}

func (a *App) GetDownloadProgress() backend.ProgressInfo {
	return backend.GetDownloadProgress()
}

func (a *App) GetDownloadQueue() backend.DownloadQueueInfo {
	return backend.GetDownloadQueue()
}

func (a *App) ClearCompletedDownloads() {
	// Stub
}

func (a *App) ClearAllDownloads() {
	backend.ClearAllDownloads()
}

func (a *App) AddToDownloadQueue(spotifyID, trackName, artistName, albumName string) string {
	backend.AddToQueue("", trackName, artistName, albumName, spotifyID)
	return spotifyID
}

func (a *App) MarkDownloadItemFailed(itemID, errorMsg string) {
	// Stub
}

func (a *App) ForceStopDownloads() {
	// Stub
}

func (a *App) ExportFailedDownloads() (string, error) {
	return "", nil // Stub
}

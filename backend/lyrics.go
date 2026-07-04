package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.senan.xyz/taglib"
)

type LRCLibResponse struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

type LyricsLine struct {
	StartTimeMs string `json:"startTimeMs"`
	Words       string `json:"words"`
	EndTimeMs   string `json:"endTimeMs"`
}

type LyricsResponse struct {
	Error    bool         `json:"error"`
	SyncType string       `json:"syncType"`
	Lines    []LyricsLine `json:"lines"`
}

type LyricsDownloadRequest struct {
	SpotifyID           string `json:"spotify_id"`
	TrackName           string `json:"track_name"`
	ArtistName          string `json:"artist_name"`
	Artists             string `json:"artists,omitempty"`
	AlbumName           string `json:"album_name"`
	AlbumArtist         string `json:"album_artist"`
	ReleaseDate         string `json:"release_date"`
	ISRC                string `json:"isrc"`
	OutputDir           string `json:"output_dir"`
	FilenameFormat      string `json:"filename_format"`
	PlaylistName        string `json:"playlist_name,omitempty"`
	Category            string `json:"category,omitempty"`
	UPC                 string `json:"upc,omitempty"`
	TrackNumber         bool   `json:"track_number"`
	Position            int    `json:"position"`
	UseAlbumTrackNumber bool   `json:"use_album_track_number"`
	DiscNumber          int    `json:"disc_number"`
	TotalTracks         int    `json:"total_tracks,omitempty"`
	TotalDiscs          int    `json:"total_discs,omitempty"`
}

type LyricsDownloadResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	File          string `json:"file,omitempty"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
}

type LyricsClient struct {
	httpClient *http.Client
}

func NewLyricsClient() *LyricsClient {
	return &LyricsClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *LyricsClient) FetchLyricsWithMetadata(trackName, artistName, albumName string, duration int) (*LyricsResponse, error) {

	apiURL := fmt.Sprintf("https://lrclib.net/api/get?artist_name=%s&track_name=%s",
		url.QueryEscape(artistName),
		url.QueryEscape(trackName))

	if albumName != "" {
		apiURL = fmt.Sprintf("%s&album_name=%s", apiURL, url.QueryEscape(albumName))
	}

	if duration > 0 {
		apiURL = fmt.Sprintf("%s&duration=%d", apiURL, duration)
	}

	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from LRCLIB: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LRCLIB returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read LRCLIB response: %v", err)
	}

	var lrcLibResp LRCLibResponse
	if err := json.Unmarshal(body, &lrcLibResp); err != nil {
		return nil, fmt.Errorf("failed to parse LRCLIB response: %v", err)
	}

	if lrcLibResp.SyncedLyrics == "" && lrcLibResp.PlainLyrics == "" {
		return nil, fmt.Errorf("LRCLIB returned empty lyrics")
	}

	return c.convertLRCLibToLyricsResponse(&lrcLibResp), nil
}

func (c *LyricsClient) convertLRCLibToLyricsResponse(lrcLib *LRCLibResponse) *LyricsResponse {
	resp := &LyricsResponse{
		Error:    false,
		SyncType: "LINE_SYNCED",
		Lines:    []LyricsLine{},
	}

	lyricsText := lrcLib.SyncedLyrics
	if lyricsText == "" {
		lyricsText = lrcLib.PlainLyrics
		resp.SyncType = "UNSYNCED"
	}

	if lyricsText == "" {
		resp.Error = true
		return resp
	}

	lines := strings.Split(lyricsText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && len(line) > 10 {
			closeBracket := strings.Index(line, "]")
			if closeBracket > 0 {
				timestamp := line[1:closeBracket]
				words := strings.TrimSpace(line[closeBracket+1:])

				ms := lrcTimestampToMs(timestamp)
				resp.Lines = append(resp.Lines, LyricsLine{
					StartTimeMs: fmt.Sprintf("%d", ms),
					Words:       words,
				})
				continue
			}
		}

		resp.Lines = append(resp.Lines, LyricsLine{
			StartTimeMs: "",
			Words:       line,
		})
	}

	return resp
}

func lrcTimestampToMs(timestamp string) int64 {
	var minutes, seconds, centiseconds int64

	n, _ := fmt.Sscanf(timestamp, "%d:%d.%d", &minutes, &seconds, &centiseconds)
	if n >= 2 {
		return minutes*60*1000 + seconds*1000 + centiseconds*10
	}
	return 0
}

func (c *LyricsClient) FetchLyricsFromLRCLibSearch(trackName, artistName string) (*LyricsResponse, error) {

	apiURL := fmt.Sprintf("https://lrclib.net/api/search?artist_name=%s&track_name=%s",
		url.QueryEscape(artistName),
		url.QueryEscape(trackName))

	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}

	var results []LRCLibResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parse failed: %v", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	var bestSynced *LRCLibResponse
	var bestPlain *LRCLibResponse
	for i := range results {
		if results[i].SyncedLyrics != "" && bestSynced == nil {
			bestSynced = &results[i]
		}
		if results[i].PlainLyrics != "" && bestPlain == nil {
			bestPlain = &results[i]
		}
		if bestSynced != nil {
			break
		}
	}

	best := bestSynced
	if best == nil {
		best = bestPlain
	}
	if best == nil {
		best = &results[0]
	}

	if best.SyncedLyrics == "" && best.PlainLyrics == "" {
		return nil, fmt.Errorf("no lyrics found in search results")
	}

	return c.convertLRCLibToLyricsResponse(best), nil
}

func simplifyTrackName(name string) string {

	if idx := strings.Index(name, "("); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}

	if idx := strings.Index(name, " - "); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}
	return name
}

func isSynced(resp *LyricsResponse) bool {
	return resp != nil && !resp.Error && resp.SyncType == "LINE_SYNCED" && len(resp.Lines) > 0
}

func hasLyrics(resp *LyricsResponse) bool {
	return resp != nil && !resp.Error && len(resp.Lines) > 0
}

func (c *LyricsClient) FetchLyricsAllSources(spotifyID, trackName, artistName, albumName string, duration int) (*LyricsResponse, string, error) {

	var unsyncedFallback *LyricsResponse
	var unsyncedSource string

	check := func(resp *LyricsResponse, err error, source string) (*LyricsResponse, string, bool) {
		if err != nil || resp == nil || resp.Error || len(resp.Lines) == 0 {
			return nil, "", false
		}
		if isSynced(resp) {
			return resp, source, true
		}

		if unsyncedFallback == nil {
			unsyncedFallback = resp
			unsyncedSource = source
		}
		return nil, "", false
	}

	var resp *LyricsResponse
	var src string
	var found bool

	resp, _ = c.FetchLyricsWithMetadata(trackName, artistName, albumName, duration)
	resp, src, found = check(resp, nil, "LRCLIB")
	if found {
		fmt.Printf("   [LRCLIB] Synced found via exact match (with album)\n")
		return resp, src, nil
	}
	fmt.Printf("   LRCLIB exact (with album): no synced\n")

	if albumName != "" {
		resp, _ = c.FetchLyricsWithMetadata(trackName, artistName, "", duration)
		resp, src, found = check(resp, nil, "LRCLIB (no album)")
		if found {
			fmt.Printf("   [LRCLIB] Synced found via exact match (no album)\n")
			return resp, src, nil
		}
		fmt.Printf("   LRCLIB exact (no album): no synced\n")
	}

	resp, _ = c.FetchLyricsFromLRCLibSearch(trackName, artistName)
	resp, src, found = check(resp, nil, "LRCLIB Search")
	if found {
		fmt.Printf("   [LRCLIB] Synced found via search\n")
		return resp, src, nil
	}
	fmt.Printf("   LRCLIB search: no synced\n")

	simplifiedTrack := simplifyTrackName(trackName)
	if simplifiedTrack != trackName {
		fmt.Printf("   Trying simplified name: %s\n", simplifiedTrack)

		resp, _ = c.FetchLyricsWithMetadata(simplifiedTrack, artistName, albumName, duration)
		resp, src, found = check(resp, nil, "LRCLIB (simplified)")
		if found {
			fmt.Printf("   [LRCLIB] Synced found via simplified exact\n")
			return resp, src, nil
		}

		resp, _ = c.FetchLyricsFromLRCLibSearch(simplifiedTrack, artistName)
		resp, src, found = check(resp, nil, "LRCLIB Search (simplified)")
		if found {
			fmt.Printf("   [LRCLIB] Synced found via simplified search\n")
			return resp, src, nil
		}
	}

	if unsyncedFallback != nil {
		fmt.Printf("   [LRCLIB] No synced found, using unsynced from: %s\n", unsyncedSource)
		return unsyncedFallback, unsyncedSource + " (unsynced)", nil
	}

	return nil, "", fmt.Errorf("lyrics not found in any source")
}

func (c *LyricsClient) ConvertToLRC(lyrics *LyricsResponse, trackName, artistName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[ti:%s]\n", trackName))
	sb.WriteString(fmt.Sprintf("[ar:%s]\n", artistName))
	sb.WriteString("[by:SpotiFlac]\n")
	sb.WriteString("\n")

	for _, line := range lyrics.Lines {
		if line.Words == "" {
			continue
		}

		if line.StartTimeMs == "" {
			sb.WriteString(fmt.Sprintf("%s\n", line.Words))
		} else {

			timestamp := msToLRCTimestamp(line.StartTimeMs)
			sb.WriteString(fmt.Sprintf("%s%s\n", timestamp, line.Words))
		}
	}

	return sb.String()
}

func msToLRCTimestamp(msStr string) string {
	var ms int64
	fmt.Sscanf(msStr, "%d", &ms)

	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	centiseconds := (ms % 1000) / 10

	return fmt.Sprintf("[%02d:%02d.%02d]", minutes, seconds, centiseconds)
}

// SearchLRCLIB fetches lyrics for the given artist and title.
func SearchLRCLIB(artist, title string) (lyrics string, synced bool, err error) {
	client := NewLyricsClient()
	
	resp, src, err := client.FetchLyricsAllSources("", title, artist, "", 0)
	if err != nil {
		return "", false, err
	}
	if resp == nil || resp.Error {
		return "", false, fmt.Errorf("lyrics not found")
	}

	isSyncedResult := isSynced(resp)
	
	var sb strings.Builder
	for _, line := range resp.Lines {
		if line.Words == "" {
			continue
		}
		if line.StartTimeMs == "" {
			sb.WriteString(fmt.Sprintf("%s\n", line.Words))
		} else {
			timestamp := msToLRCTimestamp(line.StartTimeMs)
			sb.WriteString(fmt.Sprintf("%s%s\n", timestamp, line.Words))
		}
	}
	
	fmt.Printf("[SearchLRCLIB] Lyrics found from source: %s, Synced: %v\n", src, isSyncedResult)
	return sb.String(), isSyncedResult, nil
}

// EmbedLyrics reads the tags of a file, adds the lyrics, and writes them back.
func EmbedLyrics(filePath string, lyrics string) error {
	tags, err := taglib.ReadTags(filePath)
	if err != nil {
		return fmt.Errorf("failed to read existing tags: %v", err)
	}
	
	tags[taglib.Lyrics] = []string{lyrics}
	
	if err := taglib.WriteTags(filePath, tags, taglib.Clear); err != nil {
		return fmt.Errorf("failed to write tags with lyrics: %v", err)
	}
	return nil
}



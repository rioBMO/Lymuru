package backend

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/lymuru/lymuru/backend/storage"
)

// HistoryEntry mirrors a row of the history table.
type HistoryEntry struct {
	ID        int64    `json:"id"`
	TaskID    string   `json:"task_id"`
	TaskType  string   `json:"task_type"`
	Query     string   `json:"query"`
	Status    string   `json:"status"`
	Files     []string `json:"files"`
	Error     string   `json:"error,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// HistoryResponse is the payload returned by ListHistory.
type HistoryResponse struct {
	Entries []HistoryEntry `json:"entries"`
	Total   int            `json:"total"`
}

// History manages persisted download history.
type History struct {
	db *storage.DB
}

func NewHistory(db *storage.DB) *History {
	return &History{db: db}
}

// Insert adds a new history entry. If taskID is set, it is upserted.
func (h *History) Insert(e HistoryEntry) (int64, error) {
	if h == nil || h.db == nil {
		return 0, errors.New("history: nil db")
	}
	filesJSON, _ := json.Marshal(e.Files)
	if e.CreatedAt == "" {
		e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if e.TaskID != "" {
		// Upsert by task_id.
		res, err := h.db.Conn().Exec(`
			INSERT INTO history (task_id, task_type, query, status, files, error, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(task_id) DO UPDATE SET
				status = excluded.status,
				files = excluded.files,
				error = excluded.error
		`, e.TaskID, e.TaskType, e.Query, e.Status, string(filesJSON), nullString(e.Error), e.CreatedAt)
		if err != nil {
			return 0, err
		}
		_ = res
		// Return the row id of the upserted row.
		var id int64
		if err := h.db.Conn().QueryRow(`SELECT id FROM history WHERE task_id = ?`, e.TaskID).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := h.db.Conn().Exec(`
		INSERT INTO history (task_type, query, status, files, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, e.TaskType, e.Query, e.Status, string(filesJSON), nullString(e.Error), e.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List returns history rows paginated and filtered by status and search.
// statusFilter == "" means all; search is a case-insensitive LIKE on query.
func (h *History) List(limit, offset int, statusFilter, search string) (HistoryResponse, error) {
	if h == nil || h.db == nil {
		return HistoryResponse{}, errors.New("history: nil db")
	}
	if limit <= 0 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}

	var (
		where  []string
		args   []any
		status = statusFilter
	)
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if s := strings.TrimSpace(search); s != "" {
		where = append(where, "(query LIKE ? OR task_id LIKE ?)")
		like := "%" + s + "%"
		args = append(args, like, like)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// Total count.
	var total int
	countQuery := "SELECT COUNT(*) FROM history " + whereSQL
	if err := h.db.Conn().QueryRow(countQuery, args...).Scan(&total); err != nil {
		return HistoryResponse{}, err
	}

	// Page.
	listQuery := "SELECT id, task_id, task_type, query, status, files, COALESCE(error, ''), created_at FROM history " +
		whereSQL + " ORDER BY datetime(created_at) DESC, id DESC LIMIT ? OFFSET ?"
	listArgs := append(args, limit, offset)
	rows, err := h.db.Conn().Query(listQuery, listArgs...)
	if err != nil {
		return HistoryResponse{}, err
	}
	defer rows.Close()

	var out []HistoryEntry
	for rows.Next() {
		var (
			e         HistoryEntry
			taskID    sql.NullString
			filesJSON string
			createdAt string
		)
		if err := rows.Scan(&e.ID, &taskID, &e.TaskType, &e.Query, &e.Status, &filesJSON, &e.Error, &createdAt); err != nil {
			return HistoryResponse{}, err
		}
		if taskID.Valid {
			e.TaskID = taskID.String
		}
		if filesJSON != "" {
			_ = json.Unmarshal([]byte(filesJSON), &e.Files)
		}
		e.CreatedAt = createdAt
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return HistoryResponse{}, err
	}
	return HistoryResponse{Entries: out, Total: total}, nil
}

// Delete removes a history entry by id.
func (h *History) Delete(id int64) error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(`DELETE FROM history WHERE id = ?`, id)
	return err
}

// Clear removes all history entries.
func (h *History) Clear() error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(`DELETE FROM history`)
	return err
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ---------------------------------------------------------------------------
// Download / fetch history (Spotify track download + metadata fetch records)
// ---------------------------------------------------------------------------

// DownloadHistoryItem records a completed audio download.
type DownloadHistoryItem struct {
	ID          string `json:"id"`
	SpotifyID   string `json:"spotify_id"`
	Title       string `json:"title"`
	Artists     string `json:"artists"`
	Album       string `json:"album"`
	DurationStr string `json:"duration_str"`
	CoverURL    string `json:"cover_url"`
	Quality     string `json:"quality"`
	Format      string `json:"format"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Timestamp   int64  `json:"timestamp"`
}

// FetchHistoryItem records a Spotify metadata fetch (search by URL).
type FetchHistoryItem struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Info       string `json:"info"`
	Image      string `json:"image"`
	Data       string `json:"data"`
	IsExplicit bool   `json:"is_explicit,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

// AddDownloadHistoryItem inserts a download history record.
func (h *History) AddDownloadHistoryItem(item DownloadHistoryItem) error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(
		`INSERT INTO download_history (id, spotify_id, title, artists, album, duration_str, cover_url, quality, format, path, source, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 	spotify_id = excluded.spotify_id,
		 	title = excluded.title,
		 	artists = excluded.artists,
		 	album = excluded.album,
		 	duration_str = excluded.duration_str,
		 	cover_url = excluded.cover_url,
		 	quality = excluded.quality,
		 	format = excluded.format,
		 	path = excluded.path,
		 	source = excluded.source,
		 	timestamp = excluded.timestamp`,
		item.ID, item.SpotifyID, item.Title, item.Artists, item.Album,
		item.DurationStr, item.CoverURL, item.Quality, item.Format,
		item.Path, item.Source, item.Timestamp,
	)
	return err
}

// GetDownloadHistoryItems returns all download history, newest first.
func (h *History) GetDownloadHistoryItems() ([]DownloadHistoryItem, error) {
	if h == nil || h.db == nil {
		return nil, errors.New("history: nil db")
	}
	rows, err := h.db.Conn().Query(
		`SELECT id, spotify_id, title, artists, album, duration_str, cover_url, quality, format, path, source, timestamp
		 FROM download_history ORDER BY timestamp DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DownloadHistoryItem
	for rows.Next() {
		var item DownloadHistoryItem
		if err := rows.Scan(&item.ID, &item.SpotifyID, &item.Title, &item.Artists,
			&item.Album, &item.DurationStr, &item.CoverURL, &item.Quality,
			&item.Format, &item.Path, &item.Source, &item.Timestamp); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ClearDownloadHistory deletes all download history.
func (h *History) ClearDownloadHistory() error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(`DELETE FROM download_history`)
	return err
}

// DeleteDownloadHistoryItem deletes a single download history record by ID.
func (h *History) DeleteDownloadHistoryItem(id string) error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(`DELETE FROM download_history WHERE id = ?`, id)
	return err
}

// AddFetchHistoryItem inserts a fetch history record.
func (h *History) AddFetchHistoryItem(item FetchHistoryItem) error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	isExplicit := 0
	if item.IsExplicit {
		isExplicit = 1
	}
	_, err := h.db.Conn().Exec(
		`INSERT INTO fetch_history (id, url, type, name, info, image, data, is_explicit, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.URL, item.Type, item.Name, item.Info,
		item.Image, item.Data, isExplicit, item.Timestamp,
	)
	return err
}

// GetFetchHistoryItems returns all fetch history, newest first.
func (h *History) GetFetchHistoryItems() ([]FetchHistoryItem, error) {
	if h == nil || h.db == nil {
		return nil, errors.New("history: nil db")
	}
	rows, err := h.db.Conn().Query(
		`SELECT id, url, type, name, info, image, data, is_explicit, timestamp
		 FROM fetch_history ORDER BY timestamp DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []FetchHistoryItem
	for rows.Next() {
		var item FetchHistoryItem
		var isExplicit int
		if err := rows.Scan(&item.ID, &item.URL, &item.Type, &item.Name,
			&item.Info, &item.Image, &item.Data, &isExplicit, &item.Timestamp); err != nil {
			return nil, err
		}
		item.IsExplicit = isExplicit != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

// ClearFetchHistoryByType deletes fetch history matching the given type.
func (h *History) ClearFetchHistoryByType(itemType string) error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(`DELETE FROM fetch_history WHERE type = ?`, itemType)
	return err
}

// DeleteFetchHistoryItem deletes a single fetch history record by ID.
func (h *History) DeleteFetchHistoryItem(id string) error {
	if h == nil || h.db == nil {
		return errors.New("history: nil db")
	}
	_, err := h.db.Conn().Exec(`DELETE FROM fetch_history WHERE id = ?`, id)
	return err
}

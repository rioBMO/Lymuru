package backend

import (
	"context"
	"fmt"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ProgressEvent is the payload emitted via Wails EventsEmit.
type ProgressEvent struct {
	TaskID   string  `json:"task_id"`
	Stage    string  `json:"stage"`
	Phase    string  `json:"phase"`
	Percent  float64 `json:"download_percent"`
	Bytes    int64   `json:"download_received"`
	Total    int64   `json:"download_total"`
	Query    string  `json:"query"`     // user-facing label for the task
	TaskType string  `json:"task_type"` // "search_choose" | "link" | …
}

// CompleteEvent is emitted when a task finishes successfully.
type CompleteEvent struct {
	TaskID string   `json:"task_id"`
	Files  []string `json:"files"`
}

// ErrorEvent is emitted when a task fails.
type ErrorEvent struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// StatusEvent is emitted on sidecar status changes.
type StatusEvent struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Event names — keep in sync with frontend/src/context/QueueContext.tsx.
const (
	EventTaskProgress = "task:progress"
	EventTaskComplete = "task:complete"
	EventTaskError    = "task:error"
	EventSidecar      = "sidecar:status"
)

// EmitProgress forwards a sidecar progress event into Wails runtime.
func EmitProgress(ctx context.Context, ev SidecarEvent, query, taskType string) {
	wailsruntime.EventsEmit(ctx, EventTaskProgress, ProgressEvent{
		TaskID:   ev.TaskID,
		Stage:    ev.Stage,
		Phase:    ev.Phase,
		Percent:  ev.Percent,
		Query:    query,
		TaskType: taskType,
	})
}

// EmitComplete forwards a sidecar completion event.
func EmitComplete(ctx context.Context, ev SidecarEvent) {
	wailsruntime.EventsEmit(ctx, EventTaskComplete, CompleteEvent{
		TaskID: ev.TaskID,
		Files:  ev.Files,
	})
}

// EmitError forwards a sidecar error event.
func EmitError(ctx context.Context, ev SidecarEvent) {
	wailsruntime.EventsEmit(ctx, EventTaskError, ErrorEvent{
		TaskID:  ev.TaskID,
		Message: ev.Error,
	})
}

// EmitStatus forwards a sidecar status change.
func EmitStatus(ctx context.Context, status, message string) {
	wailsruntime.EventsEmit(ctx, EventSidecar, StatusEvent{
		Status:  status,
		Message: message,
	})
}

// SidecarEventHandler wires Sidecar.OnEvent / OnStatus into the in-memory
// TaskManager and emits Wails events.
type SidecarEventHandler struct {
	Ctx     context.Context
	Tasks   *TaskManager
	History *History
}

// NewSidecarEventHandler returns a handler ready to be passed to sidecar.SetHandlers.
func NewSidecarEventHandler(ctx context.Context, tasks *TaskManager, history *History) *SidecarEventHandler {
	return &SidecarEventHandler{Ctx: ctx, Tasks: tasks, History: history}
}

// OnEvent processes a sidecar event. Safe to call concurrently.
func (h *SidecarEventHandler) OnEvent(ev SidecarEvent) {
	switch ev.Name {
	case "progress":
		if ev.TaskID != "" {
			h.Tasks.Patch(ev.TaskID, func(t *Task) {
				if ev.Stage != "" {
					t.Stage = ev.Stage
				}
				if ev.Phase != "" {
					t.Phase = ev.Phase
				}
				if ev.Percent > 0 {
					t.DownloadPercent = ev.Percent
				}
			})
		}
		// Include task metadata so the frontend can display a
		// meaningful label (artist – title, link, etc.) immediately.
		query, taskType := "", ""
		if ev.TaskID != "" {
			if t, ok := h.Tasks.Get(ev.TaskID); ok {
				query, taskType = t.Query, t.TaskType
			}
		}
		EmitProgress(h.Ctx, ev, query, taskType)
	case "complete":
		if ev.TaskID != "" {
			h.Tasks.Patch(ev.TaskID, func(t *Task) {
				t.Phase = "complete"
				t.Stage = "Complete"
				t.DownloadPercent = 100
				if len(ev.Files) > 0 {
					t.Files = ev.Files
				}
			})
			if t, ok := h.Tasks.Get(ev.TaskID); ok {
				_, _ = h.History.Insert(HistoryEntry{
					TaskID:    ev.TaskID,
					TaskType:  t.TaskType,
					Query:     t.Query,
					Status:    "completed",
					Files:     append([]string{}, t.Files...),
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
		EmitComplete(h.Ctx, ev)
	case "error":
		if ev.TaskID != "" {
			h.Tasks.Patch(ev.TaskID, func(t *Task) {
				t.Error = ev.Error
			})
			if t, ok := h.Tasks.Get(ev.TaskID); ok {
				_, _ = h.History.Insert(HistoryEntry{
					TaskID:    ev.TaskID,
					TaskType:  t.TaskType,
					Query:     t.Query,
					Status:    "failed",
					Error:     ev.Error,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
		EmitError(h.Ctx, ev)
	}
}

// OnStatus is called when the sidecar status changes.
func (h *SidecarEventHandler) OnStatus(status, message string) {
	EmitStatus(h.Ctx, status, message)
}

// String helper used elsewhere.
func _ignore() { _ = fmt.Sprint("") }

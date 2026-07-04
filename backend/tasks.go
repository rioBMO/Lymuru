package backend

import (
	"errors"
	"sort"
	"sync"
)

// Task is the in-memory representation of an in-flight or completed task.
type Task struct {
	TaskID          string   `json:"task_id"`
	TaskType        string   `json:"task_type"`
	Query           string   `json:"query"`
	Stage           string   `json:"stage"`
	Phase           string   `json:"phase"`
	DownloadPercent float64  `json:"download_percent"`
	DownloadBytes   int64    `json:"download_received"`
	DownloadTotal   int64    `json:"download_total"`
	Files           []string `json:"files"`
	Error           string   `json:"error,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

// TaskManager is a thread-safe in-memory store of active and recent tasks.
// The frontend reads from this via GetActiveTasks / GetTask bindings.
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	order []string // insertion order for stable listings
}

func NewTaskManager() *TaskManager {
	return &TaskManager{tasks: map[string]*Task{}}
}

// Add registers a new task.
func (m *TaskManager) Add(t Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.TaskID == "" {
		return
	}
	if _, ok := m.tasks[t.TaskID]; !ok {
		m.order = append(m.order, t.TaskID)
	}
	cp := t
	m.tasks[t.TaskID] = &cp
}

// Update replaces a task in place.
func (m *TaskManager) Update(t Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.TaskID == "" {
		return
	}
	cp := t
	m.tasks[t.TaskID] = &cp
}

// Patch applies a partial update to a task.
func (m *TaskManager) Patch(taskID string, fn func(t *Task)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[taskID]
	if !ok {
		return
	}
	fn(t)
}

// Get returns a copy of the task.
func (m *TaskManager) Get(taskID string) (Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[taskID]
	if !ok {
		return Task{}, false
	}
	return *t, true
}

// List returns all known tasks, most recent first.
func (m *TaskManager) List() []Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Task, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		t, ok := m.tasks[m.order[i]]
		if !ok {
			continue
		}
		out = append(out, *t)
	}
	return out
}

// ListActive returns tasks whose Phase != "complete" and Error == "".
func (m *TaskManager) ListActive() []Task {
	all := m.List()
	out := make([]Task, 0, len(all))
	for _, t := range all {
		if t.Phase != "complete" && t.Error == "" {
			out = append(out, t)
		}
	}
	return out
}

// Remove deletes a task by id.
func (m *TaskManager) Remove(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, taskID)
	for i, id := range m.order {
		if id == taskID {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
}

// Clear removes all tasks.
func (m *TaskManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = map[string]*Task{}
	m.order = nil
}

// Cancel marks a task as cancelled. The sidecar is responsible for stopping
// the underlying work; this just updates the local state.
func (m *TaskManager) Cancel(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[taskID]
	if !ok {
		return errors.New("task not found")
	}
	t.Stage = "Cancelled"
	t.Error = "Cancelled"
	return nil
}

// Count returns the number of active (non-complete, non-error) tasks.
func (m *TaskManager) Count() int {
	return len(m.ListActive())
}

// SortTasks sorts a slice of tasks by creation time, newest first.
func SortTasks(ts []Task) {
	sort.SliceStable(ts, func(i, j int) bool {
		return ts[i].CreatedAt > ts[j].CreatedAt
	})
}

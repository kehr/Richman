package analysis

import (
	"sync"
	"time"
)

// TaskStatus represents the current state of an analysis task.
type TaskStatus struct {
	TaskID    string     `json:"taskId"`
	Status    string     `json:"status"` // pending, running, completed, failed
	Progress  float64    `json:"progress"`
	Error     string     `json:"error,omitempty"`
	StartedAt time.Time  `json:"startedAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
}

// TaskStore provides in-memory tracking for analysis tasks.
type TaskStore struct {
	tasks sync.Map
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore() *TaskStore {
	return &TaskStore{}
}

// Create initializes a new task with pending status and returns it.
func (s *TaskStore) Create(taskID string) *TaskStatus {
	task := &TaskStatus{
		TaskID:    taskID,
		Status:    "pending",
		Progress:  0,
		StartedAt: time.Now(),
	}
	s.tasks.Store(taskID, task)
	return task
}

// Get retrieves the current status of a task. Returns nil if not found.
func (s *TaskStore) Get(taskID string) *TaskStatus {
	val, ok := s.tasks.Load(taskID)
	if !ok {
		return nil
	}
	task, _ := val.(*TaskStatus)
	return task
}

// UpdateProgress updates the progress of a running task.
func (s *TaskStore) UpdateProgress(taskID string, progress float64) {
	val, ok := s.tasks.Load(taskID)
	if !ok {
		return
	}
	task, _ := val.(*TaskStatus)
	task.Status = "running"
	task.Progress = progress
}

// Complete marks a task as successfully completed.
func (s *TaskStore) Complete(taskID string) {
	val, ok := s.tasks.Load(taskID)
	if !ok {
		return
	}
	task, _ := val.(*TaskStatus)
	now := time.Now()
	task.Status = "completed"
	task.Progress = 1.0
	task.DoneAt = &now
}

// Fail marks a task as failed with the given error.
func (s *TaskStore) Fail(taskID string, err error) {
	val, ok := s.tasks.Load(taskID)
	if !ok {
		return
	}
	task, _ := val.(*TaskStatus)
	now := time.Now()
	task.Status = "failed"
	task.DoneAt = &now
	if err != nil {
		task.Error = err.Error()
	}
}

package model

import "time"

// TaskStatus represents the current state of an analysis task.
type TaskStatus struct {
	TaskID    string     `json:"taskId"`
	UserID    int64      `json:"userId"`
	Status    string     `json:"status"` // pending, running, completed, failed
	Progress  float64    `json:"progress"`
	Error     string     `json:"error,omitempty"`
	StartedAt time.Time  `json:"startedAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
}

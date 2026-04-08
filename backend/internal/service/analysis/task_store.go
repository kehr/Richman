package analysis

import (
	"context"
	"sync"
	"time"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

// TaskStore provides tracking for analysis tasks with in-memory cache + DB persistence.
type TaskStore struct {
	tasks           sync.Map
	repo            *repo.AnalysisTaskRepo
	ttl             time.Duration
	cleanupInterval time.Duration
	stopCh          chan struct{}
	logger          *zap.Logger
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore(taskRepo *repo.AnalysisTaskRepo, ttl time.Duration, logger *zap.Logger) *TaskStore {
	interval := time.Hour
	if ttl > 0 && ttl < interval {
		interval = ttl / 2
		if interval <= 0 {
			interval = ttl
		}
	}
	s := &TaskStore{
		repo:            taskRepo,
		ttl:             ttl,
		cleanupInterval: interval,
		stopCh:          make(chan struct{}),
		logger:          logger,
	}
	if ttl > 0 {
		go s.cleanupLoop()
	}
	return s
}

// Stop terminates the background cleanup loop.
func (s *TaskStore) Stop() {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
}

// Create initializes a new task with pending status and returns it.
func (s *TaskStore) Create(taskID string, userID int64) *model.TaskStatus {
	task := &model.TaskStatus{
		TaskID:    taskID,
		UserID:    userID,
		Status:    "pending",
		Progress:  0,
		StartedAt: time.Now(),
	}
	s.tasks.Store(taskID, task)
	s.persist(task)
	return task
}

// Get retrieves the current status of a task. Returns nil if not found.
func (s *TaskStore) Get(taskID string) *model.TaskStatus {
	if val, ok := s.tasks.Load(taskID); ok {
		if task, ok := val.(*model.TaskStatus); ok {
			return task
		}
	}
	if s.repo == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Warn("failed to load task from repo", zap.String("task_id", taskID), zap.Error(err))
		return nil
	}
	if task != nil {
		s.tasks.Store(taskID, task)
	}
	return task
}

// UpdateProgress updates the progress of a running task.
func (s *TaskStore) UpdateProgress(taskID string, progress float64) {
	if task := s.Get(taskID); task != nil {
		task.Status = "running"
		task.Progress = progress
		s.persist(task)
	}
}

// Complete marks a task as successfully completed.
func (s *TaskStore) Complete(taskID string) {
	if task := s.Get(taskID); task != nil {
		now := time.Now()
		task.Status = "completed"
		task.Progress = 1.0
		task.DoneAt = &now
		s.persist(task)
	}
}

// Fail marks a task as failed with the given error.
func (s *TaskStore) Fail(taskID string, err error) {
	if task := s.Get(taskID); task != nil {
		now := time.Now()
		task.Status = "failed"
		task.DoneAt = &now
		if err != nil {
			task.Error = err.Error()
		}
		s.persist(task)
	}
}

func (s *TaskStore) persist(task *model.TaskStatus) {
	if s.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.repo.Upsert(ctx, task); err != nil {
		s.logger.Warn("failed to persist task status",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
	}
}

func (s *TaskStore) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-s.ttl)
			s.tasks.Range(func(key, value any) bool {
				task, ok := value.(*model.TaskStatus)
				if !ok {
					return true
				}
				if task.DoneAt != nil && task.DoneAt.Before(cutoff) {
					s.tasks.Delete(key)
				}
				return true
			})
			if s.repo != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := s.repo.DeleteOlderThan(ctx, cutoff); err != nil {
					s.logger.Warn("failed to cleanup persisted tasks", zap.Error(err))
				}
				cancel()
			}
		case <-s.stopCh:
			ticker.Stop()
			return
		}
	}
}

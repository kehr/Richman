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
	mu              sync.RWMutex // protects all TaskStatus field reads and writes
	tasks           sync.Map
	stepStartTimes  sync.Map // key: "taskID:stepKey" → time.Time
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
		Holdings:  make([]model.HoldingProgress, 0),
		Steps:     model.DefaultSteps(),
		Logs:      make([]model.TaskLog, 0),
	}
	s.tasks.Store(taskID, task)
	s.persist(task)
	return task
}

// Get retrieves a snapshot of the current task status. The returned value is safe
// for concurrent reads with ongoing task writes. Returns nil if not found.
func (s *TaskStore) Get(taskID string) *model.TaskStatus {
	s.mu.RLock()
	var snap *model.TaskStatus
	if val, ok := s.tasks.Load(taskID); ok {
		if task, ok := val.(*model.TaskStatus); ok {
			snap = copyTask(task)
		}
	}
	s.mu.RUnlock()
	if snap != nil {
		return snap
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
	s.mu.Lock()
	task := s.getTask(taskID)
	if task != nil {
		task.Status = "running"
		task.Progress = progress
	}
	s.mu.Unlock()
	if task != nil {
		s.persist(task)
	}
}

// Complete marks a task as successfully completed.
func (s *TaskStore) Complete(taskID string) {
	s.mu.Lock()
	task := s.getTask(taskID)
	if task != nil {
		now := time.Now()
		task.Status = "completed"
		task.Progress = 1.0
		task.DoneAt = &now
	}
	s.mu.Unlock()
	if task != nil {
		s.persist(task)
	}
}

// Fail marks a task as failed with the given error.
func (s *TaskStore) Fail(taskID string, err error) {
	s.mu.Lock()
	task := s.getTask(taskID)
	if task != nil {
		now := time.Now()
		task.Status = "failed"
		task.DoneAt = &now
		if err != nil {
			task.Error = err.Error()
		}
	}
	s.mu.Unlock()
	if task != nil {
		s.persist(task)
	}
}

// getTask returns the in-memory task pointer without DB fallback.
// Caller must hold s.mu before calling.
func (s *TaskStore) getTask(taskID string) *model.TaskStatus {
	if val, ok := s.tasks.Load(taskID); ok {
		if task, ok := val.(*model.TaskStatus); ok {
			return task
		}
	}
	return nil
}

// InitHoldings sets the holdings list and resets pipeline steps to pending.
func (s *TaskStore) InitHoldings(taskID string, holdings []model.HoldingProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		task.Holdings = holdings
		task.Steps = model.DefaultSteps()
	}
}

// SetCurrentHolding updates the currently analyzed holding and resets step progress.
func (s *TaskStore) SetCurrentHolding(taskID string, symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		task.CurrentHolding = symbol
		task.Steps = model.DefaultSteps()
	}
}

// UpdateHoldingStatus updates the status and result metadata for a specific holding.
func (s *TaskStore) UpdateHoldingStatus(taskID, symbol string, status model.TaskStepStatus, source, provider *string, durationMs *int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		for i := range task.Holdings {
			if task.Holdings[i].Symbol == symbol {
				task.Holdings[i].Status = status
				task.Holdings[i].SynthesisSource = source
				task.Holdings[i].ProviderUsed = provider
				task.Holdings[i].DurationMs = durationMs
				return
			}
		}
	}
}

// StartStep marks a pipeline step as running and records its start time.
func (s *TaskStore) StartStep(taskID string, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		for i := range task.Steps {
			if task.Steps[i].Key == key {
				task.Steps[i].Status = model.StepRunning
				s.stepStartTimes.Store(taskID+":"+key, time.Now())
				return
			}
		}
	}
}

// CompleteStep marks a pipeline step as done and records its duration.
func (s *TaskStore) CompleteStep(taskID string, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		for i := range task.Steps {
			if task.Steps[i].Key == key {
				task.Steps[i].Status = model.StepDone
				if v, ok := s.stepStartTimes.Load(taskID + ":" + key); ok {
					ms := time.Since(v.(time.Time)).Milliseconds()
					task.Steps[i].DurationMs = &ms
					s.stepStartTimes.Delete(taskID + ":" + key)
				}
				return
			}
		}
	}
}

// FailStep marks a pipeline step as failed and records its duration.
func (s *TaskStore) FailStep(taskID string, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		for i := range task.Steps {
			if task.Steps[i].Key == key {
				task.Steps[i].Status = model.StepFailed
				if v, ok := s.stepStartTimes.Load(taskID + ":" + key); ok {
					ms := time.Since(v.(time.Time)).Milliseconds()
					task.Steps[i].DurationMs = &ms
					s.stepStartTimes.Delete(taskID + ":" + key)
				}
				return
			}
		}
	}
}

// AppendLog appends a structured log entry to the task's log buffer.
func (s *TaskStore) AppendLog(taskID string, level model.LogLevel, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task := s.getTask(taskID); task != nil {
		task.Logs = append(task.Logs, model.TaskLog{
			Ts:    time.Now(),
			Level: level,
			Msg:   msg,
		})
	}
}

// copyTask creates a snapshot of task with independent copies of slice fields.
// Caller must hold s.mu.RLock or s.mu.Lock before calling.
func copyTask(task *model.TaskStatus) *model.TaskStatus {
	cp := *task
	cp.Holdings = append([]model.HoldingProgress(nil), task.Holdings...)
	cp.Steps = append([]model.TaskStep(nil), task.Steps...)
	cp.Logs = append([]model.TaskLog(nil), task.Logs...)
	return &cp
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
					taskID := task.TaskID
					for _, sk := range model.AllStepKeys() {
						s.stepStartTimes.Delete(taskID + ":" + sk)
					}
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

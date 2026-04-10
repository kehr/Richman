package model

import "time"

// TaskStepStatus represents the execution state of a single analysis step.
type TaskStepStatus string

// LogLevel represents the severity of a task log entry.
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

const (
	StepPending TaskStepStatus = "pending"
	StepRunning TaskStepStatus = "running"
	StepDone    TaskStepStatus = "done"
	StepFailed  TaskStepStatus = "failed"
)

// Step key constants for the five analysis pipeline stages.
const (
	StepKeyFetchData      = "fetch_data"
	StepKeyCalcIndicators = "calc_indicators"
	StepKeyRecommendation = "recommendation"
	StepKeyLLMSynthesis   = "llm_synthesis"
	StepKeyPersist        = "persist"
)

// TaskStep tracks the status and timing of one pipeline step.
type TaskStep struct {
	Key        string         `json:"key"`
	Status     TaskStepStatus `json:"status"`
	DurationMs *int64         `json:"durationMs"`
	startedAt  time.Time
}

// TaskLog holds a single structured log entry emitted during task execution.
type TaskLog struct {
	Ts    time.Time `json:"ts"`
	Level LogLevel  `json:"level"`
	Msg   string    `json:"msg"`
}

// HoldingProgress tracks per-holding analysis progress within a task.
type HoldingProgress struct {
	Symbol          string         `json:"symbol"`
	Name            string         `json:"name"`
	Status          TaskStepStatus `json:"status"`
	Progress        float64 `json:"progress"`
	SynthesisSource *string `json:"synthesisSource"`
	ProviderUsed    *string `json:"providerUsed"`
	DurationMs      *int64  `json:"durationMs"`
	startedAt       time.Time
}

// AllStepKeys returns all pipeline step keys in execution order.
func AllStepKeys() []string {
	return []string{
		StepKeyFetchData,
		StepKeyCalcIndicators,
		StepKeyRecommendation,
		StepKeyLLMSynthesis,
		StepKeyPersist,
	}
}

// DefaultSteps returns the five pipeline steps in execution order, all in pending state.
func DefaultSteps() []TaskStep {
	return []TaskStep{
		{Key: StepKeyFetchData, Status: StepPending},
		{Key: StepKeyCalcIndicators, Status: StepPending},
		{Key: StepKeyRecommendation, Status: StepPending},
		{Key: StepKeyLLMSynthesis, Status: StepPending},
		{Key: StepKeyPersist, Status: StepPending},
	}
}

// TaskStatus represents the current state of an analysis task.
type TaskStatus struct {
	TaskID         string            `json:"taskId"`
	UserID         int64             `json:"userId"`
	Status         string            `json:"status"` // pending, running, completed, failed
	Progress       float64           `json:"progress"`
	Error          string            `json:"error,omitempty"`
	StartedAt      time.Time         `json:"startedAt"`
	DoneAt         *time.Time        `json:"doneAt,omitempty"`
	CurrentHolding string            `json:"currentHolding"`
	Holdings       []HoldingProgress `json:"holdings"`
	Steps          []TaskStep        `json:"steps"`
	Logs           []TaskLog         `json:"logs"`
}

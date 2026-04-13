package model

import "time"

// EventAlert maps to rs_event_alerts. richson writes event alerts when a
// Polymarket event probability changes significantly. richman reads unalerted
// rows and marks them as alerted after sending notifications.
type EventAlert struct {
	ID              int64
	EventSlug       string
	EventTitle      string
	Source          string
	PrevProbability float64
	CurrProbability float64
	Delta           float64
	Threshold       float64
	GoldDirection   *string
	Alerted         bool
	DetectedAt      time.Time
}

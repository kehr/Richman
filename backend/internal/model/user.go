package model

import "time"

// User represents a registered user in the system.
type User struct {
	UserID       int64     `json:"userId"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	PlanID       *int64    `json:"planId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Creator      string    `json:"-"`
	Modifier     string    `json:"-"`
	IsDeleted    int16     `json:"-"`
}

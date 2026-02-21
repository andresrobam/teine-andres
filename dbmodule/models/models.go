package models

import (
	"time"

	"github.com/google/uuid"
)

type Prompt struct {
	Title     string `db:"title"`
	Prompt    string `db:"prompt"`
	LoadOrder int    `db:"load_order"`
}

type Task struct {
	ID              uuid.UUID  `db:"id"`
	ParentID        *uuid.UUID `db:"parent_task_id"`
	Title           string     `db:"title"`
	Description     *string    `db:"description"`
	CreatedAt       time.Time  `db:"created_at"`
	Status          *string    `db:"status"`
	Progress        []byte     `db:"progress"`
	StatusUpdatedAt *time.Time `db:"status_updated_at"`
	PostponedUntil  *time.Time `db:"postponed_until"`
}

type LogEntry struct {
	ID        uuid.UUID `db:"id"`
	EventType string    `db:"event_type"`
	Payload   []byte    `db:"payload"`
	CreatedAt time.Time `db:"created_at"`
}

type Credential struct {
	ID        uuid.UUID  `db:"id"`
	Name      string     `db:"name"`
	Secret    string     `db:"secret"`
	Source    *string    `db:"source"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}

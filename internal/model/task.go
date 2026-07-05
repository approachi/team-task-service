package model

import "time"

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          int64     `db:"id" json:"id"`
	TeamID      int64     `db:"team_id" json:"team_id"`
	Title       string    `db:"title" json:"title"`
	Description *string   `db:"description" json:"description,omitempty"`
	Status      Status    `db:"status" json:"status"`
	AssigneeTo  *int64    `db:"assignee_to" json:"assignee_to,omitempty"`
	CreatedBy   int64     `db:"created_by" json:"created_by"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ListFilter constrains a task listing to a single team (required — see
// docs/COVER_LETTER.md) plus optional status/assignee filters and
// pagination.
type ListFilter struct {
	TeamID     int64
	Status     *Status
	AssigneeTo *int64
	Offset     int
	Limit      int
}

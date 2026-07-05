package model

import "time"

// HistoryEntry is one row of task_history: a single changed field from a
// single task update, self-describing without needing a join back to tasks.
type HistoryEntry struct {
	ID        int64     `db:"id" json:"id"`
	TaskID    int64     `db:"task_id" json:"task_id"`
	ChangedBy int64     `db:"changed_by" json:"changed_by"`
	FieldName string    `db:"field_name" json:"field_name"`
	OldValue  *string   `db:"old_value" json:"old_value"`
	NewValue  *string   `db:"new_value" json:"new_value"`
	ChangedAt time.Time `db:"changed_at" json:"changed_at"`
}

// FieldChange describes one field changed by a task update. Produced by
// service.DiffTask (a pure function, unit-tested without a DB) and consumed
// by repository.TaskRepository.UpdateWithHistory to write both the task
// update and its audit rows in one transaction.
//
// OldValue/NewValue are text representations for the task_history audit row.
// SQLValue is the actual typed value bound into the `UPDATE tasks SET ...`
// statement (e.g. int64 or nil for assignee_to, string for title/status) —
// keeping it typed avoids relying on MySQL to coerce a stringified value
// back into the column's real type.
type FieldChange struct {
	Field    string
	OldValue *string
	NewValue *string
	SQLValue any
}

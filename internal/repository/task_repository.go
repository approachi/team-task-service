package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
)

const taskColumns = "id, team_id, title, description, status, assignee_to, created_by, created_at, updated_at"

// updatableTaskColumns whitelists the columns UpdateWithHistory may set.
// FieldChange.Field always comes from our own service.DiffTask, never from
// a request body directly, but this guards defense-in-depth against ever
// interpolating an arbitrary column name into the UPDATE statement.
var updatableTaskColumns = map[string]bool{
	"title":       true,
	"description": true,
	"status":      true,
	"assignee_to": true,
}

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *model.Task) (*model.Task, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (team_id, title, description, status, assignee_to, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		t.TeamID, t.Title, t.Description, t.Status, t.AssigneeTo, t.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get inserted task id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	var t model.Task
	query := fmt.Sprintf(`SELECT %s FROM tasks WHERE id = ?`, taskColumns)
	if err := r.db.GetContext(ctx, &t, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("task not found")
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &t, nil
}

// List applies model.ListFilter (team_id required by the caller) with
// DB-level pagination, returning the page of tasks plus the total match
// count for httpx.Meta.
func (r *TaskRepository) List(ctx context.Context, f model.ListFilter) ([]model.Task, int, error) {
	where := []string{"team_id = ?"}
	args := []any{f.TeamID}

	if f.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *f.Status)
	}
	if f.AssigneeTo != nil {
		where = append(where, "assignee_to = ?")
		args = append(args, *f.AssigneeTo)
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM tasks WHERE %s`, whereClause)
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	listArgs := append(append([]any{}, args...), f.Limit, f.Offset)
	listQuery := fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, taskColumns, whereClause)

	tasks := make([]model.Task, 0)
	if err := r.db.SelectContext(ctx, &tasks, listQuery, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, total, nil
}

// UpdateWithHistory locks the task row, derives the field changes from the
// freshly-locked row via diff, and writes one task_history row per changed
// field — atomically: a SELECT ... FOR UPDATE serializes concurrent updates
// to the same row under InnoDB's row locking, and the UPDATE + all history
// INSERTs commit or roll back together.
//
// diff is called with the row as it exists *after* the lock is acquired,
// not the caller's earlier, possibly-stale read — otherwise a concurrent
// writer that commits between the caller's read and this lock would have
// its change silently clobbered, and the history row would record a false
// old_value (the pre-lock snapshot, not the row that was actually
// overwritten).
func (r *TaskRepository) UpdateWithHistory(ctx context.Context, id int64, diff func(current *model.Task) []model.FieldChange, changedBy int64) (*model.Task, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	lockQuery := fmt.Sprintf(`SELECT %s FROM tasks WHERE id = ? FOR UPDATE`, taskColumns)
	var current model.Task
	if err := tx.GetContext(ctx, &current, lockQuery, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("task not found")
		}
		return nil, fmt.Errorf("lock task: %w", err)
	}

	changes := diff(&current)
	if len(changes) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit tx: %w", err)
		}
		return &current, nil
	}

	setClauses := make([]string, 0, len(changes))
	args := make([]any, 0, len(changes)+1)
	for _, c := range changes {
		if !updatableTaskColumns[c.Field] {
			return nil, fmt.Errorf("field %q is not updatable", c.Field)
		}
		setClauses = append(setClauses, c.Field+" = ?")
		args = append(args, c.SQLValue)
	}
	args = append(args, id)

	updateQuery := fmt.Sprintf(`UPDATE tasks SET %s WHERE id = ?`, strings.Join(setClauses, ", "))
	if _, err := tx.ExecContext(ctx, updateQuery, args...); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	for _, c := range changes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO task_history (task_id, changed_by, field_name, old_value, new_value, changed_at) VALUES (?, ?, ?, ?, ?, NOW())`,
			id, changedBy, c.Field, c.OldValue, c.NewValue,
		); err != nil {
			return nil, fmt.Errorf("insert task history: %w", err)
		}
	}

	var updated model.Task
	getQuery := fmt.Sprintf(`SELECT %s FROM tasks WHERE id = ?`, taskColumns)
	if err := tx.GetContext(ctx, &updated, getQuery, id); err != nil {
		return nil, fmt.Errorf("get updated task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &updated, nil
}

func (r *TaskRepository) ListHistory(ctx context.Context, taskID int64, offset, limit int) ([]model.HistoryEntry, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM task_history WHERE task_id = ?`, taskID); err != nil {
		return nil, 0, fmt.Errorf("count task history: %w", err)
	}

	entries := make([]model.HistoryEntry, 0)
	query := `
		SELECT id, task_id, changed_by, field_name, old_value, new_value, changed_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY changed_at ASC
		LIMIT ? OFFSET ?`
	if err := r.db.SelectContext(ctx, &entries, query, taskID, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("list task history: %w", err)
	}

	return entries, total, nil
}

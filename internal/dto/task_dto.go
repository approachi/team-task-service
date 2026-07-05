package dto

import (
	"strings"
	"time"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
)

type CreateTaskRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeTo  *int64 `json:"assignee_to,omitempty"`
}

func (r CreateTaskRequest) Validate() *apperr.Error {
	if r.TeamID <= 0 {
		return apperr.Validation("team_id", "is required")
	}
	if strings.TrimSpace(r.Title) == "" || len(r.Title) > 255 {
		return apperr.Validation("title", "is required and must be at most 255 characters")
	}
	return nil
}

// UpdateTaskRequest uses Optional* fields so the handler can tell an absent
// field apart from an explicit null (unassign) or an explicit value — see
// dto.OptionalString/OptionalInt64.
type UpdateTaskRequest struct {
	Title       OptionalString `json:"title"`
	Description OptionalString `json:"description"`
	Status      OptionalString `json:"status"`
	AssigneeTo  OptionalInt64  `json:"assignee_to"`
}

func (r UpdateTaskRequest) Validate() *apperr.Error {
	if !r.Title.Set && !r.Description.Set && !r.Status.Set && !r.AssigneeTo.Set {
		return apperr.Validation("body", "at least one field must be provided")
	}
	if r.Title.Set && (r.Title.Value == nil || strings.TrimSpace(*r.Title.Value) == "") {
		return apperr.Validation("title", "cannot be empty")
	}
	if r.Status.Set {
		if r.Status.Value == nil || !model.Status(*r.Status.Value).Valid() {
			return apperr.Validation("status", "must be one of todo, in_progress, done")
		}
	}
	return nil
}

type TaskResponse struct {
	ID          int64     `json:"id"`
	TeamID      int64     `json:"team_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status"`
	AssigneeTo  *int64    `json:"assignee_to,omitempty"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewTaskResponse(t *model.Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		TeamID:      t.TeamID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		AssigneeTo:  t.AssigneeTo,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

type HistoryEntryResponse struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	ChangedBy int64     `json:"changed_by"`
	FieldName string    `json:"field_name"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}

func NewHistoryEntryResponse(h model.HistoryEntry) HistoryEntryResponse {
	return HistoryEntryResponse{
		ID:        h.ID,
		TaskID:    h.TaskID,
		ChangedBy: h.ChangedBy,
		FieldName: h.FieldName,
		OldValue:  h.OldValue,
		NewValue:  h.NewValue,
		ChangedAt: h.ChangedAt,
	}
}

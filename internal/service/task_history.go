package service

import (
	"strconv"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
)

// DiffTask is a pure function: given the current task and the incoming
// partial update, it returns one model.FieldChange per field that actually
// changed. Unit-tested directly, with no DB involved.
func DiffTask(current *model.Task, req dto.UpdateTaskRequest) []model.FieldChange {
	var changes []model.FieldChange

	if req.Title.Set && req.Title.Value != nil && *req.Title.Value != current.Title {
		changes = append(changes, model.FieldChange{
			Field:    "title",
			OldValue: strPtr(current.Title),
			NewValue: req.Title.Value,
			SQLValue: *req.Title.Value,
		})
	}

	if req.Description.Set && !stringPtrEqual(req.Description.Value, current.Description) {
		changes = append(changes, model.FieldChange{
			Field:    "description",
			OldValue: current.Description,
			NewValue: req.Description.Value,
			SQLValue: nilableString(req.Description.Value),
		})
	}

	if req.Status.Set && req.Status.Value != nil && *req.Status.Value != string(current.Status) {
		oldStatus := string(current.Status)
		changes = append(changes, model.FieldChange{
			Field:    "status",
			OldValue: &oldStatus,
			NewValue: req.Status.Value,
			SQLValue: *req.Status.Value,
		})
	}

	if req.AssigneeTo.Set && !int64PtrEqual(req.AssigneeTo.Value, current.AssigneeTo) {
		changes = append(changes, model.FieldChange{
			Field:    "assignee_to",
			OldValue: int64PtrToStrPtr(current.AssigneeTo),
			NewValue: int64PtrToStrPtr(req.AssigneeTo.Value),
			SQLValue: nilableInt64(req.AssigneeTo.Value),
		})
	}

	return changes
}

func strPtr(s string) *string {
	return &s
}

func stringPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func nilableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func int64PtrToStrPtr(v *int64) *string {
	if v == nil {
		return nil
	}
	s := strconv.FormatInt(*v, 10)
	return &s
}

func nilableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

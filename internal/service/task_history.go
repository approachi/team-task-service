package service

import (
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

package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/service"
)

func TestDiffTask_DetectsChangedFields(t *testing.T) {
	desc := "old description"
	current := &model.Task{
		Title:       "Old title",
		Description: &desc,
		Status:      model.StatusTodo,
		AssigneeTo:  nil,
	}

	newTitle := "New title"
	newStatus := "in_progress"
	newAssignee := int64(42)

	req := dto.UpdateTaskRequest{
		Title:      dto.OptionalString{Value: &newTitle, Set: true},
		Status:     dto.OptionalString{Value: &newStatus, Set: true},
		AssigneeTo: dto.OptionalInt64{Value: &newAssignee, Set: true},
	}

	changes := service.DiffTask(current, req)
	require.Len(t, changes, 3)

	byField := make(map[string]model.FieldChange, len(changes))
	for _, c := range changes {
		byField[c.Field] = c
	}

	require.Equal(t, "New title", *byField["title"].NewValue)
	require.Equal(t, "Old title", *byField["title"].OldValue)

	require.Equal(t, "in_progress", *byField["status"].NewValue)
	require.Equal(t, "todo", *byField["status"].OldValue)

	require.Equal(t, "42", *byField["assignee_to"].NewValue)
	require.Nil(t, byField["assignee_to"].OldValue)
}

func TestDiffTask_IgnoresUnchangedFields(t *testing.T) {
	current := &model.Task{Title: "Same title", Status: model.StatusTodo}
	sameTitle := "Same title"
	req := dto.UpdateTaskRequest{Title: dto.OptionalString{Value: &sameTitle, Set: true}}

	require.Empty(t, service.DiffTask(current, req))
}

func TestDiffTask_IgnoresFieldsNotSet(t *testing.T) {
	current := &model.Task{Title: "Title", Status: model.StatusTodo}
	require.Empty(t, service.DiffTask(current, dto.UpdateTaskRequest{}))
}

func TestDiffTask_HandlesNilVsEmptyDescription(t *testing.T) {
	current := &model.Task{Description: nil}
	empty := ""
	req := dto.UpdateTaskRequest{Description: dto.OptionalString{Value: &empty, Set: true}}

	changes := service.DiffTask(current, req)
	require.Len(t, changes, 1)
	require.Equal(t, "description", changes[0].Field)
	require.Equal(t, "", *changes[0].NewValue)
}

func TestDiffTask_UnassignSetsNilSQLValue(t *testing.T) {
	assignee := int64(7)
	current := &model.Task{AssigneeTo: &assignee}
	req := dto.UpdateTaskRequest{AssigneeTo: dto.OptionalInt64{Value: nil, Set: true}}

	changes := service.DiffTask(current, req)
	require.Len(t, changes, 1)
	require.Equal(t, "assignee_to", changes[0].Field)
	require.Nil(t, changes[0].SQLValue)
	require.Equal(t, "7", *changes[0].OldValue)
	require.Nil(t, changes[0].NewValue)
}

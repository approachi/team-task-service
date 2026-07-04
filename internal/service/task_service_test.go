package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/service"
)

func TestTaskService_Create_ForbiddenForNonMember(t *testing.T) {
	teams := newFakeTeamRepo()
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	_, err := svc.Create(context.Background(), 100, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug"})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeForbidden, appErr.Code)
}

func TestTaskService_Create_RejectsNonMemberAssignee(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleMember)
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	assignee := int64(200)
	_, err := svc.Create(context.Background(), 100, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug", AssigneeTo: &assignee})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeValidation, appErr.Code)
}

func TestTaskService_Create_Success(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleMember)
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	task, err := svc.Create(context.Background(), 100, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug"})
	require.NoError(t, err)
	require.Equal(t, model.StatusTodo, task.Status)
	require.Equal(t, int64(100), task.CreatedBy)
}

func TestTaskService_Update_AdminCanManageAnyTask(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleAdmin)
	teams.setRole(1, 200, model.RoleMember)
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	assignee := int64(200)
	created, err := svc.Create(context.Background(), 200, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug", AssigneeTo: &assignee})
	require.NoError(t, err)

	newTitle := "Fixed bug"
	updated, err := svc.Update(context.Background(), 100, created.ID, dto.UpdateTaskRequest{
		Title: dto.OptionalString{Value: &newTitle, Set: true},
	})
	require.NoError(t, err)
	require.Equal(t, "Fixed bug", updated.Title)
}

func TestTaskService_Update_MemberCanUpdateOwnAssignedTask(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleMember)
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	assignee := int64(100)
	created, err := svc.Create(context.Background(), 100, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug", AssigneeTo: &assignee})
	require.NoError(t, err)

	newStatus := "in_progress"
	updated, err := svc.Update(context.Background(), 100, created.ID, dto.UpdateTaskRequest{
		Status: dto.OptionalString{Value: &newStatus, Set: true},
	})
	require.NoError(t, err)
	require.Equal(t, model.StatusInProgress, updated.Status)
}

func TestTaskService_Update_MemberForbiddenOnOthersTask(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleMember)
	teams.setRole(1, 200, model.RoleMember)
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	assignee := int64(200)
	created, err := svc.Create(context.Background(), 200, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug", AssigneeTo: &assignee})
	require.NoError(t, err)

	newStatus := "in_progress"
	_, err = svc.Update(context.Background(), 100, created.ID, dto.UpdateTaskRequest{
		Status: dto.OptionalString{Value: &newStatus, Set: true},
	})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeForbidden, appErr.Code)
}

func TestTaskService_Update_MemberCannotReassignOwnTask(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleMember)
	teams.setRole(1, 200, model.RoleMember)
	tasks := newFakeTaskRepo()
	svc := service.NewTaskService(tasks, teams)

	assignee := int64(100)
	created, err := svc.Create(context.Background(), 100, dto.CreateTaskRequest{TeamID: 1, Title: "Fix bug", AssigneeTo: &assignee})
	require.NoError(t, err)

	newAssignee := int64(200)
	_, err = svc.Update(context.Background(), 100, created.ID, dto.UpdateTaskRequest{
		AssigneeTo: dto.OptionalInt64{Value: &newAssignee, Set: true},
	})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeForbidden, appErr.Code)
}

func TestUpdateTaskRequest_InvalidStatusRejected(t *testing.T) {
	status := "bogus"
	req := dto.UpdateTaskRequest{Status: dto.OptionalString{Value: &status, Set: true}}

	err := req.Validate()
	require.Error(t, err)
	require.Equal(t, apperr.CodeValidation, err.Code)
}

func TestUpdateTaskRequest_NoFieldsSetRejected(t *testing.T) {
	err := dto.UpdateTaskRequest{}.Validate()
	require.Error(t, err)
	require.Equal(t, apperr.CodeValidation, err.Code)
}

package service

import (
	"context"
	"errors"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
)

// isNotTeamMember reports whether err is the specific apperr.Forbidden that
// TeamAuthorizer.GetRole returns for "not a member of this team", as
// distinct from any other failure (e.g. a DB timeout). Callers that probe
// membership to validate an assignee must only translate this specific case
// into a 400 validation error — anything else should propagate as-is so an
// infra failure surfaces as a 500, not a misleading "must be a member".
func isNotTeamMember(err error) bool {
	var appErr *apperr.Error
	return errors.As(err, &appErr) && appErr.Code == apperr.CodeForbidden
}

// TeamAuthorizer is the narrow slice of team membership lookup the task
// service depends on. repository.TeamRepository satisfies it structurally.
// GetRole returns an apperr.Forbidden error when userID is not a member of
// teamID — used both to authorize actions and to validate that an assignee
// belongs to the team.
type TeamAuthorizer interface {
	GetRole(ctx context.Context, teamID, userID int64) (model.Role, error)
}

type TaskService struct {
	tasks TaskRepository
	teams TeamAuthorizer
}

func NewTaskService(tasks TaskRepository, teams TeamAuthorizer) *TaskService {
	return &TaskService{tasks: tasks, teams: teams}
}

func (s *TaskService) Create(ctx context.Context, actorID int64, req dto.CreateTaskRequest) (*model.Task, error) {
	if _, err := s.teams.GetRole(ctx, req.TeamID, actorID); err != nil {
		return nil, err
	}

	if req.AssigneeTo != nil {
		if _, err := s.teams.GetRole(ctx, req.TeamID, *req.AssigneeTo); err != nil {
			if !isNotTeamMember(err) {
				return nil, err
			}
			return nil, apperr.Validation("assignee_to", "must be a member of the team")
		}
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}

	t := &model.Task{
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: description,
		Status:      model.StatusTodo,
		AssigneeTo:  req.AssigneeTo,
		CreatedBy:   actorID,
	}
	return s.tasks.Create(ctx, t)
}

func (s *TaskService) List(ctx context.Context, actorID int64, f model.ListFilter) ([]model.Task, int, error) {
	if _, err := s.teams.GetRole(ctx, f.TeamID, actorID); err != nil {
		return nil, 0, err
	}
	return s.tasks.List(ctx, f)
}

// Update enforces the role matrix: owner/admin may change any field on any
// task in the team; a member may change title/description/status only on a
// task assigned to themselves, and may never reassign a task (that always
// requires owner/admin, even on one's own task).
func (s *TaskService) Update(ctx context.Context, actorID, taskID int64, req dto.UpdateTaskRequest) (*model.Task, error) {
	current, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	actorRole, err := s.teams.GetRole(ctx, current.TeamID, actorID)
	if err != nil {
		return nil, err
	}

	isOwnTask := current.AssigneeTo != nil && *current.AssigneeTo == actorID
	if !actorRole.CanManageAnyTask() && !isOwnTask {
		return nil, apperr.Forbidden("only the assignee, owner, or admin can update this task")
	}
	if req.AssigneeTo.Set && !actorRole.CanManageAnyTask() {
		return nil, apperr.Forbidden("only owner or admin can reassign a task")
	}
	if req.AssigneeTo.Set && req.AssigneeTo.Value != nil {
		if _, err := s.teams.GetRole(ctx, current.TeamID, *req.AssigneeTo.Value); err != nil {
			if !isNotTeamMember(err) {
				return nil, err
			}
			return nil, apperr.Validation("assignee_to", "must be a member of the team")
		}
	}

	if len(DiffTask(current, req)) == 0 {
		return current, nil
	}

	// The diff is recomputed inside UpdateWithHistory against the row it
	// actually locks, not this pre-lock `current` — a concurrent writer may
	// have changed the row between this read and the lock, and the history
	// entry must record the true prior value, not this stale snapshot.
	return s.tasks.UpdateWithHistory(ctx, taskID, func(locked *model.Task) []model.FieldChange {
		return DiffTask(locked, req)
	}, actorID)
}

func (s *TaskService) GetHistory(ctx context.Context, actorID, taskID int64, offset, limit int) ([]model.HistoryEntry, int, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, 0, err
	}
	if _, err := s.teams.GetRole(ctx, task.TeamID, actorID); err != nil {
		return nil, 0, err
	}
	return s.tasks.ListHistory(ctx, taskID, offset, limit)
}

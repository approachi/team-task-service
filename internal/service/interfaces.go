package service

import (
	"context"

	"github.com/zhuk/team-task-service/internal/model"
)

// UserRepository is the subset of repository.UserRepository the service
// layer depends on. Defined here (consumer side) so unit tests can supply
// hand-written fakes instead of a real database.
type UserRepository interface {
	Create(ctx context.Context, u *model.User) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

// TeamRepository is the subset of repository.TeamRepository TeamService
// depends on.
type TeamRepository interface {
	CreateWithOwner(ctx context.Context, name string, creatorID int64) (*model.Team, error)
	ListForUser(ctx context.Context, userID int64) ([]model.TeamMembership, error)
	GetRole(ctx context.Context, teamID, userID int64) (model.Role, error)
	AddMember(ctx context.Context, teamID, userID int64, role model.Role) error
}

// TaskRepository is the subset of repository.TaskRepository TaskService
// depends on.
type TaskRepository interface {
	Create(ctx context.Context, t *model.Task) (*model.Task, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	List(ctx context.Context, f model.ListFilter) ([]model.Task, int, error)
	UpdateWithHistory(ctx context.Context, id int64, diff func(current *model.Task) []model.FieldChange, changedBy int64) (*model.Task, error)
	ListHistory(ctx context.Context, taskID int64, offset, limit int) ([]model.HistoryEntry, int, error)
}

// AnalyticsRepository is the subset of repository.AnalyticsRepository
// AnalyticsService depends on — the three ТЗ-required complex queries.
type AnalyticsRepository interface {
	TeamsSummary(ctx context.Context, callerID int64) ([]model.TeamSummary, error)
	TopCreators(ctx context.Context, callerID int64, month model.MonthRange) ([]model.TopCreator, error)
	OrphanedAssigneeTasks(ctx context.Context, callerID int64) ([]model.OrphanedAssigneeTask, error)
}

package service

import (
	"context"

	"github.com/zhuk/team-task-service/internal/model"
)

// AnalyticsService is a thin pass-through over AnalyticsRepository — these
// are read-only reports with no additional business rules, but the service
// layer is kept in front of the repository for consistency with the rest
// of the codebase (handlers never call repositories directly) and so a
// future phase can add caching or authorization nuance here without
// touching the handler.
type AnalyticsService struct {
	repo AnalyticsRepository
}

func NewAnalyticsService(repo AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) TeamsSummary(ctx context.Context, callerID int64) ([]model.TeamSummary, error) {
	return s.repo.TeamsSummary(ctx, callerID)
}

func (s *AnalyticsService) TopCreators(ctx context.Context, callerID int64, month model.MonthRange) ([]model.TopCreator, error) {
	return s.repo.TopCreators(ctx, callerID, month)
}

func (s *AnalyticsService) OrphanedAssigneeTasks(ctx context.Context, callerID int64) ([]model.OrphanedAssigneeTask, error) {
	return s.repo.OrphanedAssigneeTasks(ctx, callerID)
}

package dto

import "github.com/zhuk/team-task-service/internal/model"

type TeamSummaryResponse struct {
	TeamID             int64  `json:"team_id"`
	TeamName           string `json:"team_name"`
	MemberCount        int    `json:"member_count"`
	DoneTasksLast7Days int    `json:"done_tasks_last_7_days"`
}

func NewTeamSummaryResponse(s model.TeamSummary) TeamSummaryResponse {
	return TeamSummaryResponse{
		TeamID:             s.TeamID,
		TeamName:           s.TeamName,
		MemberCount:        s.MemberCount,
		DoneTasksLast7Days: s.DoneTasksLast7Days,
	}
}

type TopCreatorResponse struct {
	TeamID       int64  `json:"team_id"`
	TeamName     string `json:"team_name"`
	UserID       int64  `json:"user_id"`
	UserName     string `json:"user_name"`
	TasksCreated int    `json:"tasks_created"`
	Rank         int    `json:"rank"`
}

func NewTopCreatorResponse(c model.TopCreator) TopCreatorResponse {
	return TopCreatorResponse{
		TeamID:       c.TeamID,
		TeamName:     c.TeamName,
		UserID:       c.UserID,
		UserName:     c.UserName,
		TasksCreated: c.TasksCreated,
		Rank:         c.Rank,
	}
}

type OrphanedTaskResponse struct {
	TaskID        int64  `json:"task_id"`
	TeamID        int64  `json:"team_id"`
	TeamName      string `json:"team_name"`
	AssigneeID    int64  `json:"assignee_id"`
	AssigneeName  string `json:"assignee_name"`
	AssigneeEmail string `json:"assignee_email"`
}

func NewOrphanedTaskResponse(t model.OrphanedAssigneeTask) OrphanedTaskResponse {
	return OrphanedTaskResponse{
		TaskID:        t.TaskID,
		TeamID:        t.TeamID,
		TeamName:      t.TeamName,
		AssigneeID:    t.AssigneeID,
		AssigneeName:  t.AssigneeName,
		AssigneeEmail: t.AssigneeEmail,
	}
}

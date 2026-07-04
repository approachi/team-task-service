package model

import "time"

// TeamSummary is one row of the "per-team JOIN + aggregation" report:
// team name, member count, and done-tasks-in-the-last-7-days count.
type TeamSummary struct {
	TeamID             int64  `db:"team_id" json:"team_id"`
	TeamName           string `db:"team_name" json:"team_name"`
	MemberCount        int    `db:"member_count" json:"member_count"`
	DoneTasksLast7Days int    `db:"done_tasks_last_7_days" json:"done_tasks_last_7_days"`
}

// TopCreator is one row of the "top-3 task creators per team per month"
// window-function report.
type TopCreator struct {
	TeamID       int64  `db:"team_id" json:"team_id"`
	TeamName     string `db:"team_name" json:"team_name"`
	UserID       int64  `db:"user_id" json:"user_id"`
	UserName     string `db:"user_name" json:"user_name"`
	TasksCreated int    `db:"tasks_created" json:"tasks_created"`
	Rank         int    `db:"rnk" json:"rank"`
}

// MonthRange is a [Start, End) half-open interval covering one calendar
// month, used to bound the top-creators report.
type MonthRange struct {
	Start time.Time
	End   time.Time
}

// MonthRangeFor returns the [start, end) interval for the calendar month
// containing t.
func MonthRangeFor(t time.Time) MonthRange {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return MonthRange{Start: start, End: start.AddDate(0, 1, 0)}
}

// OrphanedAssigneeTask is one row of the referential-integrity check: a
// task whose assignee is not (or no longer) a member of the task's team.
type OrphanedAssigneeTask struct {
	TaskID        int64  `db:"task_id" json:"task_id"`
	TeamID        int64  `db:"team_id" json:"team_id"`
	TeamName      string `db:"team_name" json:"team_name"`
	AssigneeID    int64  `db:"assignee_id" json:"assignee_id"`
	AssigneeName  string `db:"assignee_name" json:"assignee_name"`
	AssigneeEmail string `db:"assignee_email" json:"assignee_email"`
}

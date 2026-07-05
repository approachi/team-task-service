// Package repository — analytics_repository.go implements the three
// "complex SQL" queries required by the ТЗ: a multi-table JOIN with
// aggregation, a window-function report, and a referential-integrity
// check. See docs/COVER_LETTER.md for the reasoning behind the technology
// choices used to implement each query.
package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/zhuk/team-task-service/internal/model"
)

type AnalyticsRepository struct {
	db *sqlx.DB
}

func NewAnalyticsRepository(db *sqlx.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// TeamsSummary answers: "for each team the caller belongs to — its name,
// member count, and how many tasks it moved to 'done' in the last 7 days."
//
// team_members and tasks are both joined against teams (3 tables), which
// creates a row fan-out (member rows × task rows per team); COUNT(DISTINCT
// ...) on each side collapses that fan-out back to the correct counts
// instead of double-counting. idx_tasks_team_status_updated (added in
// Phase 1 specifically for this query) covers the done/updated_at filter.
func (r *AnalyticsRepository) TeamsSummary(ctx context.Context, callerID int64) ([]model.TeamSummary, error) {
	const query = `
		SELECT
			t.id   AS team_id,
			t.name AS team_name,
			COUNT(DISTINCT tm.user_id) AS member_count,
			COUNT(DISTINCT CASE
				WHEN tk.status = 'done' AND tk.updated_at >= (NOW() - INTERVAL 7 DAY)
				THEN tk.id
			END) AS done_tasks_last_7_days
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks tk ON tk.team_id = t.id
		WHERE t.id IN (SELECT team_id FROM team_members WHERE user_id = ?)
		GROUP BY t.id, t.name
		ORDER BY t.name`

	summaries := make([]model.TeamSummary, 0)
	if err := r.db.SelectContext(ctx, &summaries, query, callerID); err != nil {
		return nil, fmt.Errorf("teams summary: %w", err)
	}
	return summaries, nil
}

// TopCreators answers: "top 3 users by tasks created, per team, within
// [month.Start, month.End)" using ROW_NUMBER() OVER (PARTITION BY team_id
// ORDER BY tasks_created DESC) — the required window-function query. A CTE
// keeps the per-user counting step and the ranking step readable as two
// separate stages instead of nesting subqueries.
func (r *AnalyticsRepository) TopCreators(ctx context.Context, callerID int64, month model.MonthRange) ([]model.TopCreator, error) {
	const query = `
		WITH counts AS (
			SELECT
				tk.team_id,
				tk.created_by AS user_id,
				COUNT(*) AS tasks_created
			FROM tasks tk
			WHERE tk.created_at >= ? AND tk.created_at < ?
			GROUP BY tk.team_id, tk.created_by
		),
		ranked AS (
			SELECT
				counts.*,
				ROW_NUMBER() OVER (
					PARTITION BY team_id ORDER BY tasks_created DESC, user_id ASC
				) AS rnk
			FROM counts
		)
		SELECT
			ranked.team_id,
			t.name AS team_name,
			ranked.user_id,
			u.name AS user_name,
			ranked.tasks_created,
			ranked.rnk
		FROM ranked
		JOIN teams t ON t.id = ranked.team_id
		JOIN users u ON u.id = ranked.user_id
		WHERE ranked.rnk <= 3
		  AND ranked.team_id IN (SELECT team_id FROM team_members WHERE user_id = ?)
		ORDER BY ranked.team_id, ranked.rnk`

	creators := make([]model.TopCreator, 0)
	if err := r.db.SelectContext(ctx, &creators, query, month.Start, month.End, callerID); err != nil {
		return nil, fmt.Errorf("top creators: %w", err)
	}
	return creators, nil
}

// OrphanedAssigneeTasks answers: "find tasks whose assignee is not a member
// of the task's team" — a referential-integrity check via NOT EXISTS
// against team_members. In normal operation this can never happen (both
// task creation and reassignment validate team membership at write time —
// see TaskService), so this exists to catch anomalies from, e.g., a future
// "remove team member" endpoint that doesn't also touch their tasks.
// Scoped to the caller's own teams, same as the other two reports, so it
// doesn't leak other teams' assignment data to an unrelated authenticated
// user.
func (r *AnalyticsRepository) OrphanedAssigneeTasks(ctx context.Context, callerID int64) ([]model.OrphanedAssigneeTask, error) {
	const query = `
		SELECT
			tk.id AS task_id,
			tk.team_id,
			t.name AS team_name,
			tk.assignee_to AS assignee_id,
			u.name AS assignee_name,
			u.email AS assignee_email
		FROM tasks tk
		JOIN teams t ON t.id = tk.team_id
		JOIN users u ON u.id = tk.assignee_to
		WHERE tk.assignee_to IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM team_members tm
			WHERE tm.team_id = tk.team_id AND tm.user_id = tk.assignee_to
		  )
		  AND tk.team_id IN (SELECT team_id FROM team_members WHERE user_id = ?)
		ORDER BY tk.id`

	tasks := make([]model.OrphanedAssigneeTask, 0)
	if err := r.db.SelectContext(ctx, &tasks, query, callerID); err != nil {
		return nil, fmt.Errorf("orphaned assignee tasks: %w", err)
	}
	return tasks, nil
}

//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/repository"
	"github.com/zhuk/team-task-service/test/integration/testhelper"
)

// TestAnalyticsRepository_Integration exercises the three ТЗ-required
// complex queries against a real MySQL: the JOIN+aggregation team summary,
// the window-function top-creators report, and the referential-integrity
// orphaned-assignee check. One container is shared across all three
// sub-tests (distinct fixtures per sub-test) to avoid three separate
// container start-ups.
func TestAnalyticsRepository_Integration(t *testing.T) {
	db := testhelper.StartMySQL(t, "../../migrations")
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)

	t.Run("TeamsSummary", func(t *testing.T) {
		owner, err := userRepo.Create(ctx, &model.User{Email: "ts-owner@example.com", PasswordHash: "hash", Name: "Owner"})
		require.NoError(t, err)
		member, err := userRepo.Create(ctx, &model.User{Email: "ts-member@example.com", PasswordHash: "hash", Name: "Member"})
		require.NoError(t, err)

		team, err := teamRepo.CreateWithOwner(ctx, "Summary Team", owner.ID)
		require.NoError(t, err)
		require.NoError(t, teamRepo.AddMember(ctx, team.ID, member.ID, model.RoleMember))

		recentDone, err := taskRepo.Create(ctx, &model.Task{TeamID: team.ID, Title: "Recent done", Status: model.StatusTodo, CreatedBy: owner.ID})
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `UPDATE tasks SET status = 'done', updated_at = NOW() WHERE id = ?`, recentDone.ID)
		require.NoError(t, err)

		oldDone, err := taskRepo.Create(ctx, &model.Task{TeamID: team.ID, Title: "Old done", Status: model.StatusTodo, CreatedBy: owner.ID})
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `UPDATE tasks SET status = 'done', updated_at = ? WHERE id = ?`,
			time.Now().AddDate(0, 0, -10), oldDone.ID)
		require.NoError(t, err)

		_, err = taskRepo.Create(ctx, &model.Task{TeamID: team.ID, Title: "Still todo", Status: model.StatusTodo, CreatedBy: owner.ID})
		require.NoError(t, err)

		summaries, err := analyticsRepo.TeamsSummary(ctx, owner.ID)
		require.NoError(t, err)
		require.Len(t, summaries, 1)
		require.Equal(t, team.ID, summaries[0].TeamID)
		require.Equal(t, 2, summaries[0].MemberCount)
		require.Equal(t, 1, summaries[0].DoneTasksLast7Days)
	})

	t.Run("TopCreators", func(t *testing.T) {
		owner, err := userRepo.Create(ctx, &model.User{Email: "tc-owner@example.com", PasswordHash: "hash", Name: "Owner"})
		require.NoError(t, err)
		team, err := teamRepo.CreateWithOwner(ctx, "Creators Team", owner.ID)
		require.NoError(t, err)

		userA, err := userRepo.Create(ctx, &model.User{Email: "tc-a@example.com", PasswordHash: "hash", Name: "Alice"})
		require.NoError(t, err)
		userB, err := userRepo.Create(ctx, &model.User{Email: "tc-b@example.com", PasswordHash: "hash", Name: "Bob"})
		require.NoError(t, err)
		userC, err := userRepo.Create(ctx, &model.User{Email: "tc-c@example.com", PasswordHash: "hash", Name: "Carol"})
		require.NoError(t, err)
		userD, err := userRepo.Create(ctx, &model.User{Email: "tc-d@example.com", PasswordHash: "hash", Name: "Dave"})
		require.NoError(t, err)

		createN := func(userID int64, n int) {
			for i := 0; i < n; i++ {
				_, err := taskRepo.Create(ctx, &model.Task{TeamID: team.ID, Title: "Task", Status: model.StatusTodo, CreatedBy: userID})
				require.NoError(t, err)
			}
		}
		createN(userA.ID, 5)
		createN(userB.ID, 3)
		createN(userC.ID, 2)
		createN(userD.ID, 1)

		month := model.MonthRangeFor(time.Now())
		creators, err := analyticsRepo.TopCreators(ctx, owner.ID, month)
		require.NoError(t, err)
		require.Len(t, creators, 3, "must cut off at top 3, excluding Dave")

		require.Equal(t, userA.ID, creators[0].UserID)
		require.Equal(t, 5, creators[0].TasksCreated)
		require.Equal(t, 1, creators[0].Rank)

		require.Equal(t, userB.ID, creators[1].UserID)
		require.Equal(t, 3, creators[1].TasksCreated)
		require.Equal(t, 2, creators[1].Rank)

		require.Equal(t, userC.ID, creators[2].UserID)
		require.Equal(t, 2, creators[2].TasksCreated)
		require.Equal(t, 3, creators[2].Rank)
	})

	t.Run("OrphanedAssigneeTasks", func(t *testing.T) {
		owner, err := userRepo.Create(ctx, &model.User{Email: "oa-owner@example.com", PasswordHash: "hash", Name: "Owner"})
		require.NoError(t, err)
		exMember, err := userRepo.Create(ctx, &model.User{Email: "oa-exmember@example.com", PasswordHash: "hash", Name: "ExMember"})
		require.NoError(t, err)

		team, err := teamRepo.CreateWithOwner(ctx, "Orphan Team", owner.ID)
		require.NoError(t, err)
		require.NoError(t, teamRepo.AddMember(ctx, team.ID, exMember.ID, model.RoleMember))

		task, err := taskRepo.Create(ctx, &model.Task{
			TeamID: team.ID, Title: "Assigned then orphaned", Status: model.StatusTodo,
			AssigneeTo: &exMember.ID, CreatedBy: owner.ID,
		})
		require.NoError(t, err)

		// No "remove team member" endpoint exists yet, so the anomaly this
		// check exists to catch is set up directly against the DB.
		_, err = db.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = ? AND user_id = ?`, team.ID, exMember.ID)
		require.NoError(t, err)

		orphaned, err := analyticsRepo.OrphanedAssigneeTasks(ctx, owner.ID)
		require.NoError(t, err)
		require.Len(t, orphaned, 1)
		require.Equal(t, task.ID, orphaned[0].TaskID)
		require.Equal(t, exMember.ID, orphaned[0].AssigneeID)
	})
}

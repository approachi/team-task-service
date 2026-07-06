//go:build integration

package repository_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/repository"
	"github.com/zhuk/team-task-service/internal/repository/testhelper"
)

// TestTaskRepository_Integration proves the harness end-to-end against a
// real MySQL: migrations apply cleanly, the UpdateWithHistory transaction
// is atomic and produces one task_history row per changed field, and the
// team_members unique constraint is enforced by MySQL itself (not just
// application code).
func TestTaskRepository_Integration(t *testing.T) {
	db := testhelper.StartMySQL(t, "../../migrations")
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	owner, err := userRepo.Create(ctx, &model.User{Email: "owner@example.com", PasswordHash: "hash", Name: "Owner"})
	require.NoError(t, err)

	assignee, err := userRepo.Create(ctx, &model.User{Email: "assignee@example.com", PasswordHash: "hash", Name: "Assignee"})
	require.NoError(t, err)

	otherAssignee, err := userRepo.Create(ctx, &model.User{Email: "other@example.com", PasswordHash: "hash", Name: "Other"})
	require.NoError(t, err)

	team, err := teamRepo.CreateWithOwner(ctx, "Team Alpha", owner.ID)
	require.NoError(t, err)

	require.NoError(t, teamRepo.AddMember(ctx, team.ID, assignee.ID, model.RoleMember))
	require.NoError(t, teamRepo.AddMember(ctx, team.ID, otherAssignee.ID, model.RoleMember))

	task, err := taskRepo.Create(ctx, &model.Task{
		TeamID:     team.ID,
		Title:      "Fix bug",
		Status:     model.StatusTodo,
		AssigneeTo: &assignee.ID,
		CreatedBy:  owner.ID,
	})
	require.NoError(t, err)
	require.Equal(t, model.StatusTodo, task.Status)

	changes := []model.FieldChange{
		{
			Field:    "status",
			OldValue: strPtr(string(model.StatusTodo)),
			NewValue: strPtr(string(model.StatusInProgress)),
			SQLValue: string(model.StatusInProgress),
		},
		{
			Field:    "assignee_to",
			OldValue: strPtr(strconv.FormatInt(assignee.ID, 10)),
			NewValue: strPtr(strconv.FormatInt(otherAssignee.ID, 10)),
			SQLValue: otherAssignee.ID,
		},
	}

	updated, err := taskRepo.UpdateWithHistory(ctx, task.ID, func(_ *model.Task) []model.FieldChange {
		return changes
	}, owner.ID)
	require.NoError(t, err)
	require.Equal(t, model.StatusInProgress, updated.Status)
	require.NotNil(t, updated.AssigneeTo)
	require.Equal(t, otherAssignee.ID, *updated.AssigneeTo)

	history, total, err := taskRepo.ListHistory(ctx, task.ID, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, history, 2)

	byField := make(map[string]model.HistoryEntry, len(history))
	for _, h := range history {
		byField[h.FieldName] = h
	}

	require.Equal(t, "todo", *byField["status"].OldValue)
	require.Equal(t, "in_progress", *byField["status"].NewValue)
	require.Equal(t, owner.ID, byField["status"].ChangedBy)

	require.Equal(t, strconv.FormatInt(assignee.ID, 10), *byField["assignee_to"].OldValue)
	require.Equal(t, strconv.FormatInt(otherAssignee.ID, 10), *byField["assignee_to"].NewValue)

	// A duplicate (team_id, user_id) membership must be rejected by MySQL's
	// unique constraint, not merely by application-level checks.
	err = teamRepo.AddMember(ctx, team.ID, assignee.ID, model.RoleMember)
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeConflict, appErr.Code)
}

// TestTaskRepository_UpdateWithHistory_ConcurrentUpdatesDoNotCorruptAudit
// proves the fix for the stale-diff bug: two goroutines race to change the
// same task's status, each deriving its FieldChange from the row
// UpdateWithHistory hands them (post-lock), not a snapshot read before the
// race started. If the diff were computed from a pre-lock read instead, the
// loser's history row would record a false old_value (whatever it read
// before the winner committed) instead of the value the winner actually
// left behind — this test would fail on that bug.
func TestTaskRepository_UpdateWithHistory_ConcurrentUpdatesDoNotCorruptAudit(t *testing.T) {
	db := testhelper.StartMySQL(t, "../../migrations")
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	owner, err := userRepo.Create(ctx, &model.User{Email: "race-owner@example.com", PasswordHash: "hash", Name: "Owner"})
	require.NoError(t, err)

	team, err := teamRepo.CreateWithOwner(ctx, "Team Race", owner.ID)
	require.NoError(t, err)

	task, err := taskRepo.Create(ctx, &model.Task{
		TeamID:    team.ID,
		Title:     "Race me",
		Status:    model.StatusTodo,
		CreatedBy: owner.ID,
	})
	require.NoError(t, err)

	toStatus := func(target model.Status) func(current *model.Task) []model.FieldChange {
		return func(current *model.Task) []model.FieldChange {
			if current.Status == target {
				return nil
			}
			old := string(current.Status)
			newVal := string(target)
			return []model.FieldChange{{
				Field:    "status",
				OldValue: &old,
				NewValue: &newVal,
				SQLValue: newVal,
			}}
		}
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, _ = taskRepo.UpdateWithHistory(ctx, task.ID, toStatus(model.StatusInProgress), owner.ID)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _ = taskRepo.UpdateWithHistory(ctx, task.ID, toStatus(model.StatusDone), owner.ID)
	}()
	close(start)
	wg.Wait()

	history, total, err := taskRepo.ListHistory(ctx, task.ID, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total, "both concurrent writers must each produce exactly one history row")

	// Regardless of which goroutine's UPDATE actually won the row lock
	// first, the history rows form a chain: the first row's old_value must
	// be the task's true initial state, and the second row's old_value
	// must equal the first row's new_value — never a stale pre-lock read.
	require.Equal(t, string(model.StatusTodo), *history[0].OldValue)
	require.Equal(t, *history[0].NewValue, *history[1].OldValue,
		"second writer's recorded old_value must match what the first writer actually left behind, "+
			"not a snapshot read before either writer acquired the row lock")

	finalTask, err := taskRepo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, *history[1].NewValue, string(finalTask.Status),
		"the task's final status must match the last history entry's new_value")
}

func strPtr(s string) *string {
	return &s
}

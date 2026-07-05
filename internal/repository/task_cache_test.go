package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/repository"
)

// fakeTaskDelegate is a hand-written fake for the unexported taskDelegate
// interface CachingTaskRepository wraps — passing it in from this external
// test package works because Go interface satisfaction is structural.
type fakeTaskDelegate struct {
	listCalls int
	tasks     []model.Task
	total     int
	updateFn  func(ctx context.Context, id int64, diff func(current *model.Task) []model.FieldChange, changedBy int64) (*model.Task, error)
}

func (f *fakeTaskDelegate) Create(_ context.Context, t *model.Task) (*model.Task, error) {
	return t, nil
}

func (f *fakeTaskDelegate) GetByID(_ context.Context, id int64) (*model.Task, error) {
	return &model.Task{ID: id}, nil
}

func (f *fakeTaskDelegate) List(_ context.Context, _ model.ListFilter) ([]model.Task, int, error) {
	f.listCalls++
	return f.tasks, f.total, nil
}

func (f *fakeTaskDelegate) UpdateWithHistory(ctx context.Context, id int64, diff func(current *model.Task) []model.FieldChange, changedBy int64) (*model.Task, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, id, diff, changedBy)
	}
	return &model.Task{ID: id}, nil
}

func (f *fakeTaskDelegate) ListHistory(_ context.Context, _ int64, _, _ int) ([]model.HistoryEntry, int, error) {
	return nil, 0, nil
}

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestCachingTaskRepository_ListCachesResult(t *testing.T) {
	client := newTestRedisClient(t)
	delegate := &fakeTaskDelegate{tasks: []model.Task{{ID: 1, Title: "Fix bug"}}, total: 1}
	repo := repository.NewCachingTaskRepository(delegate, client, time.Minute)
	filter := model.ListFilter{TeamID: 1, Offset: 0, Limit: 20}

	tasks, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, tasks, 1)
	require.Equal(t, 1, delegate.listCalls)

	tasks2, total2, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, total, total2)
	require.Equal(t, tasks, tasks2)
	require.Equal(t, 1, delegate.listCalls, "second List should be served from cache, not the delegate")
}

func TestCachingTaskRepository_CreateInvalidatesCache(t *testing.T) {
	client := newTestRedisClient(t)
	delegate := &fakeTaskDelegate{tasks: []model.Task{{ID: 1}}, total: 1}
	repo := repository.NewCachingTaskRepository(delegate, client, time.Minute)
	filter := model.ListFilter{TeamID: 1, Offset: 0, Limit: 20}

	_, _, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 1, delegate.listCalls)

	delegate.tasks = []model.Task{{ID: 1}, {ID: 2}}
	delegate.total = 2
	_, err = repo.Create(context.Background(), &model.Task{TeamID: 1, Title: "New task"})
	require.NoError(t, err)

	_, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 2, total, "must reflect the new task, not a stale cached count")
	require.Equal(t, 2, delegate.listCalls, "cache must be bypassed after the version bump")
}

func TestCachingTaskRepository_UpdateInvalidatesCache(t *testing.T) {
	client := newTestRedisClient(t)
	delegate := &fakeTaskDelegate{tasks: []model.Task{{ID: 1, Status: model.StatusTodo}}, total: 1}
	repo := repository.NewCachingTaskRepository(delegate, client, time.Minute)
	filter := model.ListFilter{TeamID: 1, Offset: 0, Limit: 20}

	_, _, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 1, delegate.listCalls)

	delegate.updateFn = func(_ context.Context, id int64, _ func(current *model.Task) []model.FieldChange, _ int64) (*model.Task, error) {
		return &model.Task{ID: id, TeamID: 1, Status: model.StatusInProgress}, nil
	}
	_, err = repo.UpdateWithHistory(context.Background(), 1, func(_ *model.Task) []model.FieldChange {
		return []model.FieldChange{{Field: "status"}}
	}, 1)
	require.NoError(t, err)

	delegate.tasks = []model.Task{{ID: 1, Status: model.StatusInProgress}}
	_, _, err = repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 2, delegate.listCalls, "cache must be invalidated by the version bump on update")
}

func TestCachingTaskRepository_DifferentFiltersDoNotShareCacheEntries(t *testing.T) {
	client := newTestRedisClient(t)
	delegate := &fakeTaskDelegate{tasks: []model.Task{{ID: 1}}, total: 1}
	repo := repository.NewCachingTaskRepository(delegate, client, time.Minute)

	todo := model.StatusTodo
	done := model.StatusDone

	_, _, err := repo.List(context.Background(), model.ListFilter{TeamID: 1, Status: &todo, Offset: 0, Limit: 20})
	require.NoError(t, err)
	_, _, err = repo.List(context.Background(), model.ListFilter{TeamID: 1, Status: &done, Offset: 0, Limit: 20})
	require.NoError(t, err)

	require.Equal(t, 2, delegate.listCalls, "distinct filters must use distinct cache keys")
}

func TestCachingTaskRepository_RedisDownFallsBackToDelegate(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	delegate := &fakeTaskDelegate{tasks: []model.Task{{ID: 1}}, total: 1}
	repo := repository.NewCachingTaskRepository(delegate, client, time.Minute)

	mr.Close()

	tasks, total, err := repo.List(context.Background(), model.ListFilter{TeamID: 1, Offset: 0, Limit: 20})
	require.NoError(t, err, "a cache outage must not fail the request")
	require.Equal(t, 1, total)
	require.Len(t, tasks, 1)
	require.Equal(t, 1, delegate.listCalls)
}

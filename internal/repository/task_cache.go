package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zhuk/team-task-service/internal/model"
)

// TaskListCacheTTL is the fixed TTL for cached team task lists, per ТЗ
// requirement ("Кеширование в Redis: список задач команды (TTL 5 мин)").
const TaskListCacheTTL = 5 * time.Minute

// taskDelegate is the method set CachingTaskRepository wraps. Declared here
// (not imported from the service package) so the repository package never
// depends on the service package — the caching decorator still ends up
// satisfying service.TaskRepository structurally, since Go interface
// satisfaction only cares about the method set, not the declared type name.
type taskDelegate interface {
	Create(ctx context.Context, t *model.Task) (*model.Task, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	List(ctx context.Context, f model.ListFilter) ([]model.Task, int, error)
	UpdateWithHistory(ctx context.Context, id int64, diff func(current *model.Task) []model.FieldChange, changedBy int64) (*model.Task, error)
	ListHistory(ctx context.Context, taskID int64, offset, limit int) ([]model.HistoryEntry, int, error)
}

// CachingTaskRepository decorates a TaskRepository with a cache-aside layer
// over List — the "team task list" the ТЗ asks to cache. GetByID and
// ListHistory pass straight through: only the list endpoint is in scope.
//
// Invalidation is version-based rather than key-deletion-based: each team
// has a version counter in Redis, and every cache key embeds the team's
// current version. Create/UpdateWithHistory bump the counter (a single
// INCR) instead of scanning for and deleting every cached filter/page
// combination for that team — cheaper than a KEYS/SCAN sweep, and stale
// entries simply age out via TaskListCacheTTL instead of needing an
// explicit delete.
type CachingTaskRepository struct {
	next  taskDelegate
	cache *redis.Client
	ttl   time.Duration
}

func NewCachingTaskRepository(next taskDelegate, cache *redis.Client, ttl time.Duration) *CachingTaskRepository {
	return &CachingTaskRepository{next: next, cache: cache, ttl: ttl}
}

func (c *CachingTaskRepository) Create(ctx context.Context, t *model.Task) (*model.Task, error) {
	created, err := c.next.Create(ctx, t)
	if err != nil {
		return nil, err
	}
	c.bumpVersion(ctx, created.TeamID)
	return created, nil
}

func (c *CachingTaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	return c.next.GetByID(ctx, id)
}

type taskListCacheEntry struct {
	Tasks []model.Task `json:"tasks"`
	Total int          `json:"total"`
}

func (c *CachingTaskRepository) List(ctx context.Context, f model.ListFilter) ([]model.Task, int, error) {
	key, err := c.listCacheKey(ctx, f)
	if err == nil {
		if entry, hit := c.getCachedList(ctx, key); hit {
			return entry.Tasks, entry.Total, nil
		}
	}

	tasks, total, err := c.next.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	if key != "" {
		c.setCachedList(ctx, key, taskListCacheEntry{Tasks: tasks, Total: total})
	}

	return tasks, total, nil
}

func (c *CachingTaskRepository) UpdateWithHistory(ctx context.Context, id int64, diff func(current *model.Task) []model.FieldChange, changedBy int64) (*model.Task, error) {
	updated, err := c.next.UpdateWithHistory(ctx, id, diff, changedBy)
	if err != nil {
		return nil, err
	}
	c.bumpVersion(ctx, updated.TeamID)
	return updated, nil
}

func (c *CachingTaskRepository) ListHistory(ctx context.Context, taskID int64, offset, limit int) ([]model.HistoryEntry, int, error) {
	return c.next.ListHistory(ctx, taskID, offset, limit)
}

func (c *CachingTaskRepository) listCacheKey(ctx context.Context, f model.ListFilter) (string, error) {
	version, err := c.cache.Get(ctx, c.versionKey(f.TeamID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			version = "0"
		} else {
			// Redis unreachable: signal "no key" so List() falls back to
			// the DB instead of failing the request over a cache outage.
			return "", err
		}
	}

	status := "any"
	if f.Status != nil {
		status = string(*f.Status)
	}
	assignee := "any"
	if f.AssigneeTo != nil {
		assignee = strconv.FormatInt(*f.AssigneeTo, 10)
	}

	return fmt.Sprintf("tasks:list:team:%d:v:%s:status:%s:assignee:%s:offset:%d:limit:%d",
		f.TeamID, version, status, assignee, f.Offset, f.Limit), nil
}

func (c *CachingTaskRepository) versionKey(teamID int64) string {
	return fmt.Sprintf("tasks:list:version:%d", teamID)
}

func (c *CachingTaskRepository) bumpVersion(ctx context.Context, teamID int64) {
	if err := c.cache.Incr(ctx, c.versionKey(teamID)).Err(); err != nil {
		slog.Warn("bump task list cache version failed; stale entries will still expire via TTL",
			"error", err, "team_id", teamID)
	}
}

func (c *CachingTaskRepository) getCachedList(ctx context.Context, key string) (taskListCacheEntry, bool) {
	raw, err := c.cache.Get(ctx, key).Bytes()
	if err != nil {
		return taskListCacheEntry{}, false
	}
	var entry taskListCacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		slog.Warn("decode cached task list failed", "error", err)
		return taskListCacheEntry{}, false
	}
	return entry, true
}

func (c *CachingTaskRepository) setCachedList(ctx context.Context, key string, entry taskListCacheEntry) {
	raw, err := json.Marshal(entry)
	if err != nil {
		slog.Warn("encode task list for cache failed", "error", err)
		return
	}
	if err := c.cache.Set(ctx, key, raw, c.ttl).Err(); err != nil {
		slog.Warn("write task list cache failed", "error", err)
	}
}

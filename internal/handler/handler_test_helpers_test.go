// Shared test scaffolding for the handler package's HTTP-level tests:
// in-memory fakes for the repository interfaces each service depends on,
// a router builder that wires the real Auth middleware (so "missing/invalid
// token" behavior is exercised end-to-end instead of assumed), and small
// request/response helpers.
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
	"github.com/zhuk/team-task-service/internal/platform/jwtauth"
	"github.com/zhuk/team-task-service/internal/platform/middleware"
)

const testJWTSecret = "handler-test-secret"

// dataEnvelope mirrors httpx.Envelope but with a concrete Data type so
// tests can decode a success response directly instead of round-tripping
// through `any`.
type dataEnvelope[T any] struct {
	Data T           `json:"data"`
	Meta *httpx.Meta `json:"meta,omitempty"`
}

// issueToken returns a valid bearer token for userID, signed with the same
// secret the test router's Auth middleware validates against.
func issueToken(t *testing.T, userID int64) string {
	t.Helper()
	token, _, err := jwtauth.Issue([]byte(testJWTSecret), time.Hour, userID, "actor@example.com")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

// newAuthedRouter builds a chi router with the real Auth middleware wired
// in front of mount, mirroring router.New's protected route group.
func newAuthedRouter(mount func(r chi.Router)) http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth([]byte(testJWTSecret)))
		mount(r)
	})
	return r
}

func newRequest(method, path, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	r.Header.Set("Content-Type", "application/json")
	return r
}

func newAuthedRequest(t *testing.T, method, path, body string, userID int64) *http.Request {
	t.Helper()
	r := newRequest(method, path, body)
	r.Header.Set("Authorization", "Bearer "+issueToken(t, userID))
	return r
}

func do(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeInto(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("decode response body: %v (body=%s)", err, rec.Body.String())
	}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorEnvelope {
	t.Helper()
	var env httpx.ErrorEnvelope
	decodeInto(t, rec, &env)
	return env
}

// --- fakeUserRepo: service.UserRepository ---

type fakeUserRepo struct {
	byEmail map[string]*model.User
	nextID  int64
}

// nextID starts well above the small hardcoded actorID/teamID constants
// used throughout these tests, so a seeded user's ID never accidentally
// collides with the acting user's ID.
func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: make(map[string]*model.User), nextID: 100}
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) (*model.User, error) {
	if _, exists := f.byEmail[u.Email]; exists {
		return nil, apperr.Conflict("email is already registered")
	}
	f.nextID++
	stored := *u
	stored.ID = f.nextID
	f.byEmail[u.Email] = &stored
	return &stored, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, apperr.NotFound("user not found")
	}
	return u, nil
}

// seed registers a user directly, bypassing HTTP registration, and returns
// the assigned ID.
func (f *fakeUserRepo) seed(email, name string) int64 {
	f.nextID++
	f.byEmail[email] = &model.User{ID: f.nextID, Email: email, Name: name}
	return f.nextID
}

// --- fakeTeamRepo: service.TeamRepository / service.TeamAuthorizer ---

type teamUserKey struct {
	teamID int64
	userID int64
}

type fakeTeamRepo struct {
	roles             map[teamUserKey]model.Role
	members           map[int64]map[int64]bool
	membershipsByUser map[int64][]model.TeamMembership
	createdTeamID     int64
}

func newFakeTeamRepo() *fakeTeamRepo {
	return &fakeTeamRepo{
		roles:             make(map[teamUserKey]model.Role),
		members:           make(map[int64]map[int64]bool),
		membershipsByUser: make(map[int64][]model.TeamMembership),
	}
}

func (f *fakeTeamRepo) CreateWithOwner(_ context.Context, name string, creatorID int64) (*model.Team, error) {
	f.createdTeamID++
	return &model.Team{ID: f.createdTeamID, Name: name, CreatedBy: creatorID}, nil
}

func (f *fakeTeamRepo) ListForUser(_ context.Context, userID int64) ([]model.TeamMembership, error) {
	return f.membershipsByUser[userID], nil
}

func (f *fakeTeamRepo) GetRole(_ context.Context, teamID, userID int64) (model.Role, error) {
	role, ok := f.roles[teamUserKey{teamID, userID}]
	if !ok {
		return "", apperr.Forbidden("not a member of this team")
	}
	return role, nil
}

func (f *fakeTeamRepo) AddMember(_ context.Context, teamID, userID int64, role model.Role) error {
	if f.members[teamID] == nil {
		f.members[teamID] = make(map[int64]bool)
	}
	if f.members[teamID][userID] {
		return apperr.Conflict("user is already a member of this team")
	}
	f.members[teamID][userID] = true
	f.roles[teamUserKey{teamID, userID}] = role
	return nil
}

// setRole seeds a membership directly, bypassing AddMember's conflict check.
func (f *fakeTeamRepo) setRole(teamID, userID int64, role model.Role) {
	f.roles[teamUserKey{teamID, userID}] = role
}

func (f *fakeTeamRepo) setMemberships(userID int64, memberships []model.TeamMembership) {
	f.membershipsByUser[userID] = memberships
}

// --- fakeTaskRepo: service.TaskRepository ---

type fakeTaskRepo struct {
	tasks         map[int64]*model.Task
	nextID        int64
	historyResult []model.HistoryEntry
	historyTotal  int
}

func newFakeTaskRepo() *fakeTaskRepo {
	return &fakeTaskRepo{tasks: make(map[int64]*model.Task)}
}

func (f *fakeTaskRepo) Create(_ context.Context, t *model.Task) (*model.Task, error) {
	f.nextID++
	stored := *t
	stored.ID = f.nextID
	f.tasks[stored.ID] = &stored
	return &stored, nil
}

func (f *fakeTaskRepo) GetByID(_ context.Context, id int64) (*model.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, apperr.NotFound("task not found")
	}
	return t, nil
}

func (f *fakeTaskRepo) List(_ context.Context, filter model.ListFilter) ([]model.Task, int, error) {
	out := make([]model.Task, 0)
	for _, t := range f.tasks {
		if t.TeamID == filter.TeamID {
			out = append(out, *t)
		}
	}
	return out, len(out), nil
}

// UpdateWithHistory applies FieldChange.SQLValue the same way the real
// repository's UPDATE statement would, so tests exercise the same typed
// values service.DiffTask produces.
func (f *fakeTaskRepo) UpdateWithHistory(_ context.Context, id int64, diff func(current *model.Task) []model.FieldChange, _ int64) (*model.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, apperr.NotFound("task not found")
	}
	updated := *t
	for _, c := range diff(t) {
		switch c.Field {
		case "title":
			updated.Title = c.SQLValue.(string)
		case "description":
			if c.SQLValue == nil {
				updated.Description = nil
			} else {
				s := c.SQLValue.(string)
				updated.Description = &s
			}
		case "status":
			updated.Status = model.Status(c.SQLValue.(string))
		case "assignee_to":
			if c.SQLValue == nil {
				updated.AssigneeTo = nil
			} else {
				v := c.SQLValue.(int64)
				updated.AssigneeTo = &v
			}
		}
	}
	f.tasks[id] = &updated
	return &updated, nil
}

func (f *fakeTaskRepo) ListHistory(_ context.Context, _ int64, _, _ int) ([]model.HistoryEntry, int, error) {
	return f.historyResult, f.historyTotal, nil
}

// seed inserts a task directly, bypassing HTTP creation, and returns the
// assigned ID.
func (f *fakeTaskRepo) seed(task model.Task) int64 {
	f.nextID++
	task.ID = f.nextID
	f.tasks[task.ID] = &task
	return task.ID
}

// --- fakeNotifier: notify.Notifier ---

type fakeNotifier struct {
	err error
}

func (f *fakeNotifier) SendInvite(_ context.Context, _ string, _ int64) error {
	return f.err
}

// --- fakeAnalyticsRepo: service.AnalyticsRepository ---

type fakeAnalyticsRepo struct {
	teamsSummary  []model.TeamSummary
	topCreators   []model.TopCreator
	orphaned      []model.OrphanedAssigneeTask
	capturedMonth model.MonthRange
}

func (f *fakeAnalyticsRepo) TeamsSummary(_ context.Context, _ int64) ([]model.TeamSummary, error) {
	return f.teamsSummary, nil
}

func (f *fakeAnalyticsRepo) TopCreators(_ context.Context, _ int64, month model.MonthRange) ([]model.TopCreator, error) {
	f.capturedMonth = month
	return f.topCreators, nil
}

func (f *fakeAnalyticsRepo) OrphanedAssigneeTasks(_ context.Context, _ int64) ([]model.OrphanedAssigneeTask, error) {
	return f.orphaned, nil
}

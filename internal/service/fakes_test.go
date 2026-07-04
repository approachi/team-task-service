package service_test

import (
	"context"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
)

// fakeUserRepo, fakeTeamRepo, fakeTaskRepo, and fakeNotifier are
// hand-written fakes for the narrow interfaces service depends on
// (service.UserRepository, service.TeamRepository, service.TaskRepository,
// notify.Notifier) so these tests exercise business logic without a DB.

type fakeUserRepo struct {
	byEmail map[string]*model.User
	nextID  int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: make(map[string]*model.User)}
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

type teamUserKey struct {
	teamID int64
	userID int64
}

type fakeTeamRepo struct {
	roles   map[teamUserKey]model.Role
	members map[int64]map[int64]bool
}

func newFakeTeamRepo() *fakeTeamRepo {
	return &fakeTeamRepo{
		roles:   make(map[teamUserKey]model.Role),
		members: make(map[int64]map[int64]bool),
	}
}

func (f *fakeTeamRepo) CreateWithOwner(_ context.Context, name string, creatorID int64) (*model.Team, error) {
	return &model.Team{ID: 1, Name: name, CreatedBy: creatorID}, nil
}

func (f *fakeTeamRepo) ListForUser(_ context.Context, _ int64) ([]model.TeamMembership, error) {
	return nil, nil
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

// setRole seeds a membership directly, bypassing AddMember's conflict check
// — used by tests to set up the actor's role before calling the service.
func (f *fakeTeamRepo) setRole(teamID, userID int64, role model.Role) {
	f.roles[teamUserKey{teamID, userID}] = role
}

type fakeTaskRepo struct {
	tasks  map[int64]*model.Task
	nextID int64
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
// values service.DiffTask produces. diff is invoked against the fake's
// stored copy, mirroring how the real repository calls it against the
// row it just locked.
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
	return nil, 0, nil
}

type fakeNotifier struct {
	calls int
	err   error
}

func (f *fakeNotifier) SendInvite(_ context.Context, _ string, _ int64) error {
	f.calls++
	return f.err
}

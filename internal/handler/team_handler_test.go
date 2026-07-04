package handler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/handler"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/service"
)

var errFakeNotifyDown = errors.New("notify down")

func newTeamTestRouter(notifier *fakeNotifier) (http.Handler, *fakeTeamRepo, *fakeUserRepo) {
	teams := newFakeTeamRepo()
	users := newFakeUserRepo()
	teamSvc := service.NewTeamService(teams, users, notifier)
	h := handler.NewTeamHandler(teamSvc)

	router := newAuthedRouter(func(r chi.Router) {
		r.Post("/teams", h.Create)
		r.Get("/teams", h.List)
		r.Post("/teams/{id}/invite", h.Invite)
	})
	return router, teams, users
}

func TestTeamHandler_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, _, _ := newTeamTestRouter(&fakeNotifier{})
		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams", `{"name":"Platform"}`, actorID))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[dto.TeamResponse]
		decodeInto(t, rec, &env)
		if env.Data.Name != "Platform" || env.Data.CreatedBy != actorID {
			t.Fatalf("unexpected team in response: %+v", env.Data)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		router, _, _ := newTeamTestRouter(&fakeNotifier{})
		rec := do(router, newRequest(http.MethodPost, "/teams", `{"name":"Platform"}`))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		router, _, _ := newTeamTestRouter(&fakeNotifier{})
		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams", `{`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("validation failure: empty name", func(t *testing.T) {
		router, _, _ := newTeamTestRouter(&fakeNotifier{})
		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams", `{"name":"  "}`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestTeamHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, teams, _ := newTeamTestRouter(&fakeNotifier{})
		teams.setMemberships(actorID, []model.TeamMembership{
			{ID: teamID, Name: "Platform", Role: model.RoleOwner},
		})

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/teams", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.TeamMembershipResponse]
		decodeInto(t, rec, &env)
		if len(env.Data) != 1 || env.Data[0].Role != model.RoleOwner {
			t.Fatalf("unexpected memberships in response: %+v", env.Data)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		router, _, _ := newTeamTestRouter(&fakeNotifier{})
		rec := do(router, newRequest(http.MethodGet, "/teams", ""))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestTeamHandler_Invite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, teams, users := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleOwner)
		inviteeID := users.seed("invitee@example.com", "Invitee")

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{"email":"invitee@example.com"}`, actorID))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[dto.InviteResponse]
		decodeInto(t, rec, &env)
		if env.Data.TeamID != teamID || env.Data.UserID != inviteeID || env.Data.Role != model.RoleMember {
			t.Fatalf("unexpected invite response: %+v", env.Data)
		}
	})

	t.Run("invalid id in path", func(t *testing.T) {
		router, teams, _ := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleOwner)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/abc/invite", `{"email":"x@example.com"}`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		router, teams, _ := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleOwner)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		router, teams, _ := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleOwner)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{"email":"not-an-email"}`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("forbidden for member", func(t *testing.T) {
		router, teams, _ := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleMember)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{"email":"x@example.com"}`, actorID))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("unknown email", func(t *testing.T) {
		router, teams, _ := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleOwner)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{"email":"ghost@example.com"}`, actorID))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("already a member", func(t *testing.T) {
		router, teams, users := newTeamTestRouter(&fakeNotifier{})
		teams.setRole(teamID, actorID, model.RoleOwner)
		inviteeID := users.seed("already@example.com", "Already Member")
		teams.setRole(teamID, inviteeID, model.RoleMember)
		teams.members[teamID] = map[int64]bool{inviteeID: true}

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{"email":"already@example.com"}`, actorID))

		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("notifier failure does not fail the request", func(t *testing.T) {
		router, teams, users := newTeamTestRouter(&fakeNotifier{err: errFakeNotifyDown})
		teams.setRole(teamID, actorID, model.RoleOwner)
		users.seed("resilient@example.com", "Resilient")

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/teams/"+itoa(teamID)+"/invite", `{"email":"resilient@example.com"}`, actorID))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 despite notifier failure (body=%s)", rec.Code, rec.Body.String())
		}
	})
}

package handler_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/handler"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/service"
)

func newTaskTestRouter() (http.Handler, *fakeTaskRepo, *fakeTeamRepo) {
	tasks := newFakeTaskRepo()
	teams := newFakeTeamRepo()
	taskSvc := service.NewTaskService(tasks, teams)
	h := handler.NewTaskHandler(taskSvc)

	router := newAuthedRouter(func(r chi.Router) {
		r.Post("/tasks", h.Create)
		r.Get("/tasks", h.List)
		r.Put("/tasks/{id}", h.Update)
		r.Get("/tasks/{id}/history", h.History)
	})
	return router, tasks, teams
}

const (
	actorID = int64(1)
	teamID  = int64(10)
)

func TestTaskHandler_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)

		body := `{"team_id":10,"title":"Write tests"}`
		rec := do(router, newAuthedRequest(t, http.MethodPost, "/tasks", body, actorID))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[dto.TaskResponse]
		decodeInto(t, rec, &env)
		if env.Data.Title != "Write tests" || env.Data.Status != "todo" || env.Data.TeamID != teamID {
			t.Fatalf("unexpected task in response: %+v", env.Data)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		router, _, _ := newTaskTestRouter()
		rec := do(router, newRequest(http.MethodPost, "/tasks", `{"team_id":10,"title":"x"}`))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/tasks", `{`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("validation failure: missing title", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/tasks", `{"team_id":10,"title":""}`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("forbidden for non-member", func(t *testing.T) {
		router, _, _ := newTaskTestRouter()
		// actor has no role seeded for teamID at all.
		rec := do(router, newAuthedRequest(t, http.MethodPost, "/tasks", `{"team_id":10,"title":"x"}`, actorID))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects assignee outside the team", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)
		// assignee (id 99) is never given a role in the team.

		rec := do(router, newAuthedRequest(t, http.MethodPost, "/tasks", `{"team_id":10,"title":"x","assignee_to":99}`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
		}
		env := decodeError(t, rec)
		if _, ok := env.Error.Details["assignee_to"]; !ok {
			t.Fatalf("expected assignee_to validation detail, got %+v", env.Error.Details)
		}
	})
}

func TestTaskHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)
		tasks.seed(model.Task{TeamID: teamID, Title: "A", Status: model.StatusTodo, CreatedBy: actorID})
		tasks.seed(model.Task{TeamID: teamID, Title: "B", Status: model.StatusTodo, CreatedBy: actorID})
		tasks.seed(model.Task{TeamID: teamID + 1, Title: "Other team", Status: model.StatusTodo, CreatedBy: actorID})

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks?team_id=10", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.TaskResponse]
		decodeInto(t, rec, &env)
		if len(env.Data) != 2 {
			t.Fatalf("got %d tasks, want 2 (only this team's tasks)", len(env.Data))
		}
		if env.Meta == nil || env.Meta.Total != 2 || env.Meta.Page != 1 || env.Meta.PageSize != 20 {
			t.Fatalf("unexpected meta: %+v", env.Meta)
		}
	})

	t.Run("page_size is capped at the max", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks?team_id=10&page_size=500", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.TaskResponse]
		decodeInto(t, rec, &env)
		if env.Meta.PageSize != 100 {
			t.Fatalf("page_size = %d, want capped to 100", env.Meta.PageSize)
		}
	})

	t.Run("forbidden for non-member", func(t *testing.T) {
		router, _, _ := newTaskTestRouter()
		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks?team_id=10", "", actorID))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("query validation", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)

		cases := map[string]string{
			"missing team_id":       "/tasks",
			"non-numeric team_id":   "/tasks?team_id=abc",
			"zero team_id":          "/tasks?team_id=0",
			"invalid status":        "/tasks?team_id=10&status=bogus",
			"non-numeric assignee":  "/tasks?team_id=10&assignee_to=abc",
			"non-numeric page":      "/tasks?team_id=10&page=abc",
			"non-numeric page_size": "/tasks?team_id=10&page_size=abc",
		}
		for name, path := range cases {
			t.Run(name, func(t *testing.T) {
				rec := do(router, newAuthedRequest(t, http.MethodGet, path, "", actorID))
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
				}
			})
		}
	})
}

func TestTaskHandler_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleAdmin)
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "Old title", Status: model.StatusTodo, CreatedBy: actorID})

		rec := do(router, newAuthedRequest(t, http.MethodPut, "/tasks/"+itoa(taskID), `{"title":"New title"}`, actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[dto.TaskResponse]
		decodeInto(t, rec, &env)
		if env.Data.Title != "New title" {
			t.Fatalf("title = %q, want %q", env.Data.Title, "New title")
		}
	})

	t.Run("invalid id in path", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleAdmin)

		for _, id := range []string{"abc", "0", "-1"} {
			t.Run(id, func(t *testing.T) {
				rec := do(router, newAuthedRequest(t, http.MethodPut, "/tasks/"+id, `{"title":"x"}`, actorID))
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400", rec.Code)
				}
			})
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleAdmin)
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "T", Status: model.StatusTodo})

		rec := do(router, newAuthedRequest(t, http.MethodPut, "/tasks/"+itoa(taskID), `{`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("validation failure: empty update body", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleAdmin)
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "T", Status: model.StatusTodo})

		rec := do(router, newAuthedRequest(t, http.MethodPut, "/tasks/"+itoa(taskID), `{}`, actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("forbidden: member updating another member's task", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		otherUserID := int64(2)
		teams.setRole(teamID, actorID, model.RoleMember)
		teams.setRole(teamID, otherUserID, model.RoleMember)
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "T", Status: model.StatusTodo, AssigneeTo: &otherUserID})

		rec := do(router, newAuthedRequest(t, http.MethodPut, "/tasks/"+itoa(taskID), `{"title":"hijacked"}`, actorID))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body=%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		router, _, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleAdmin)

		rec := do(router, newAuthedRequest(t, http.MethodPut, "/tasks/999", `{"title":"x"}`, actorID))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (body=%s)", rec.Code, rec.Body.String())
		}
	})
}

func TestTaskHandler_History(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "T", Status: model.StatusTodo})
		tasks.historyResult = []model.HistoryEntry{{ID: 1, TaskID: taskID, FieldName: "title", ChangedBy: actorID}}
		tasks.historyTotal = 1

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks/"+itoa(taskID)+"/history", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.HistoryEntryResponse]
		decodeInto(t, rec, &env)
		if len(env.Data) != 1 || env.Data[0].FieldName != "title" {
			t.Fatalf("unexpected history in response: %+v", env.Data)
		}
		if env.Meta == nil || env.Meta.Total != 1 {
			t.Fatalf("unexpected meta: %+v", env.Meta)
		}
	})

	t.Run("page_size is capped at the max", func(t *testing.T) {
		router, tasks, teams := newTaskTestRouter()
		teams.setRole(teamID, actorID, model.RoleMember)
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "T", Status: model.StatusTodo})

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks/"+itoa(taskID)+"/history?page_size=1000", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.HistoryEntryResponse]
		decodeInto(t, rec, &env)
		if env.Meta.PageSize != 100 {
			t.Fatalf("page_size = %d, want capped to 100", env.Meta.PageSize)
		}
	})

	t.Run("invalid id in path", func(t *testing.T) {
		router, _, _ := newTaskTestRouter()
		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks/abc/history", "", actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("forbidden for non-member", func(t *testing.T) {
		router, tasks, _ := newTaskTestRouter()
		taskID := tasks.seed(model.Task{TeamID: teamID, Title: "T", Status: model.StatusTodo})

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks/"+itoa(taskID)+"/history", "", actorID))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		router, _, _ := newTaskTestRouter()
		rec := do(router, newAuthedRequest(t, http.MethodGet, "/tasks/999/history", "", actorID))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
	})
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

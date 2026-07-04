package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/handler"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/service"
)

func newReportTestRouter() (http.Handler, *fakeAnalyticsRepo) {
	analytics := &fakeAnalyticsRepo{}
	analyticsSvc := service.NewAnalyticsService(analytics)
	h := handler.NewReportHandler(analyticsSvc)

	router := newAuthedRouter(func(r chi.Router) {
		r.Get("/reports/teams-summary", h.TeamsSummary)
		r.Get("/reports/top-creators", h.TopCreators)
		r.Get("/reports/orphaned-assignees", h.OrphanedAssignees)
	})
	return router, analytics
}

func TestReportHandler_TeamsSummary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, analytics := newReportTestRouter()
		analytics.teamsSummary = []model.TeamSummary{
			{TeamID: teamID, TeamName: "Platform", MemberCount: 3, DoneTasksLast7Days: 5},
		}

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/reports/teams-summary", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.TeamSummaryResponse]
		decodeInto(t, rec, &env)
		if len(env.Data) != 1 || env.Data[0].MemberCount != 3 {
			t.Fatalf("unexpected teams summary in response: %+v", env.Data)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		router, _ := newReportTestRouter()
		rec := do(router, newRequest(http.MethodGet, "/reports/teams-summary", ""))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestReportHandler_TopCreators(t *testing.T) {
	t.Run("defaults to the current month when unset", func(t *testing.T) {
		router, analytics := newReportTestRouter()
		want := model.MonthRangeFor(time.Now())

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/reports/top-creators", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		if !analytics.capturedMonth.Start.Equal(want.Start) || !analytics.capturedMonth.End.Equal(want.End) {
			t.Fatalf("month range = %+v, want %+v", analytics.capturedMonth, want)
		}
	})

	t.Run("explicit month is parsed", func(t *testing.T) {
		router, analytics := newReportTestRouter()

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/reports/top-creators?month=2026-01", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		wantStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		wantEnd := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
		if !analytics.capturedMonth.Start.Equal(wantStart) || !analytics.capturedMonth.End.Equal(wantEnd) {
			t.Fatalf("month range = %+v, want [%v, %v)", analytics.capturedMonth, wantStart, wantEnd)
		}
	})

	t.Run("invalid month format", func(t *testing.T) {
		router, _ := newReportTestRouter()
		rec := do(router, newAuthedRequest(t, http.MethodGet, "/reports/top-creators?month=not-a-month", "", actorID))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		router, _ := newReportTestRouter()
		rec := do(router, newRequest(http.MethodGet, "/reports/top-creators", ""))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestReportHandler_OrphanedAssignees(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router, analytics := newReportTestRouter()
		analytics.orphaned = []model.OrphanedAssigneeTask{
			{TaskID: 1, TeamID: teamID, TeamName: "Platform", AssigneeID: 5, AssigneeName: "Ghost", AssigneeEmail: "ghost@example.com"},
		}

		rec := do(router, newAuthedRequest(t, http.MethodGet, "/reports/orphaned-assignees", "", actorID))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var env dataEnvelope[[]dto.OrphanedTaskResponse]
		decodeInto(t, rec, &env)
		if len(env.Data) != 1 || env.Data[0].AssigneeEmail != "ghost@example.com" {
			t.Fatalf("unexpected orphaned tasks in response: %+v", env.Data)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		router, _ := newReportTestRouter()
		rec := do(router, newRequest(http.MethodGet, "/reports/orphaned-assignees", ""))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

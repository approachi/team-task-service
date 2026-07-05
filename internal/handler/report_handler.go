package handler

import (
	"net/http"
	"time"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
	"github.com/zhuk/team-task-service/internal/service"
)

const monthParamLayout = "2006-01"

type ReportHandler struct {
	analytics *service.AnalyticsService
}

func NewReportHandler(analytics *service.AnalyticsService) *ReportHandler {
	return &ReportHandler{analytics: analytics}
}

// TeamsSummary godoc
// @Summary      Per-team member count and done-tasks-in-last-7-days, for the caller's teams
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} httpx.Envelope{data=[]dto.TeamSummaryResponse}
// @Failure      401 {object} httpx.ErrorEnvelope
// @Router       /reports/teams-summary [get]
func (h *ReportHandler) TeamsSummary(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	summaries, err := h.analytics.TeamsSummary(r.Context(), actorID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	resp := make([]dto.TeamSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		resp = append(resp, dto.NewTeamSummaryResponse(s))
	}
	httpx.WriteData(w, http.StatusOK, resp)
}

// TopCreators godoc
// @Summary      Top-3 task creators per team for a given month (default: current month)
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        month query string false "YYYY-MM, defaults to the current month"
// @Success      200 {object} httpx.Envelope{data=[]dto.TopCreatorResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      401 {object} httpx.ErrorEnvelope
// @Router       /reports/top-creators [get]
func (h *ReportHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	month, err := parseMonthParam(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	creators, err := h.analytics.TopCreators(r.Context(), actorID, month)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	resp := make([]dto.TopCreatorResponse, 0, len(creators))
	for _, c := range creators {
		resp = append(resp, dto.NewTopCreatorResponse(c))
	}
	httpx.WriteData(w, http.StatusOK, resp)
}

// OrphanedAssignees godoc
// @Summary      Integrity check: tasks whose assignee is not a member of the task's team
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} httpx.Envelope{data=[]dto.OrphanedTaskResponse}
// @Failure      401 {object} httpx.ErrorEnvelope
// @Router       /reports/orphaned-assignees [get]
func (h *ReportHandler) OrphanedAssignees(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	tasks, err := h.analytics.OrphanedAssigneeTasks(r.Context(), actorID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	resp := make([]dto.OrphanedTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, dto.NewOrphanedTaskResponse(t))
	}
	httpx.WriteData(w, http.StatusOK, resp)
}

func parseMonthParam(r *http.Request) (model.MonthRange, error) {
	raw := r.URL.Query().Get("month")
	if raw == "" {
		return model.MonthRangeFor(time.Now()), nil
	}

	parsed, err := time.Parse(monthParamLayout, raw)
	if err != nil {
		return model.MonthRange{}, apperr.Validation("month", "must be in YYYY-MM format")
	}
	return model.MonthRangeFor(parsed), nil
}

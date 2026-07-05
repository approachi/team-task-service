package handler

import (
	"net/http"
	"strconv"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
	"github.com/zhuk/team-task-service/internal/service"
)

const (
	defaultTaskPageSize    = 20
	maxTaskPageSize        = 100
	defaultHistoryPageSize = 50
	maxHistoryPageSize     = 100
)

type TaskHandler struct {
	tasks *service.TaskService
}

func NewTaskHandler(tasks *service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

// Create godoc
// @Summary      Create a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateTaskRequest true "Task"
// @Success      201 {object} httpx.Envelope{data=dto.TaskResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      403 {object} httpx.ErrorEnvelope
// @Router       /tasks [post]
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req dto.CreateTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	task, err := h.tasks.Create(r.Context(), actorID, req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteData(w, http.StatusCreated, dto.NewTaskResponse(task))
}

// List godoc
// @Summary      List tasks in a team, filtered and paginated
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        team_id     query int    true  "Team ID"
// @Param        status      query string false "todo|in_progress|done"
// @Param        assignee_to query int    false "Assignee user ID"
// @Param        page        query int    false "Page number (default 1)"
// @Param        page_size   query int    false "Page size (default 20, max 100)"
// @Success      200 {object} httpx.Envelope{data=[]dto.TaskResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      403 {object} httpx.ErrorEnvelope
// @Router       /tasks [get]
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	filter, err := parseListFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	page, err := httpx.ParsePageRequest(r, defaultTaskPageSize, maxTaskPageSize)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	filter.Offset = page.Offset()
	filter.Limit = page.PageSize

	tasks, total, err := h.tasks.List(r.Context(), actorID, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	resp := make([]dto.TaskResponse, 0, len(tasks))
	for i := range tasks {
		resp = append(resp, dto.NewTaskResponse(&tasks[i]))
	}
	httpx.WriteList(w, http.StatusOK, resp, httpx.NewMeta(page, total))
}

// Update godoc
// @Summary      Update a task (partial)
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Task ID"
// @Param        request body dto.UpdateTaskRequest true "Fields to update"
// @Success      200 {object} httpx.Envelope{data=dto.TaskResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      403 {object} httpx.ErrorEnvelope
// @Failure      404 {object} httpx.ErrorEnvelope
// @Router       /tasks/{id} [put]
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	taskID, err := parseIDParam(r, "id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req dto.UpdateTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	task, err := h.tasks.Update(r.Context(), actorID, taskID, req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteData(w, http.StatusOK, dto.NewTaskResponse(task))
}

// History godoc
// @Summary      Get a task's audit history
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        id        path  int true  "Task ID"
// @Param        page      query int false "Page number (default 1)"
// @Param        page_size query int false "Page size (default 50, max 100)"
// @Success      200 {object} httpx.Envelope{data=[]dto.HistoryEntryResponse}
// @Failure      403 {object} httpx.ErrorEnvelope
// @Failure      404 {object} httpx.ErrorEnvelope
// @Router       /tasks/{id}/history [get]
func (h *TaskHandler) History(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	taskID, err := parseIDParam(r, "id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	page, err := httpx.ParsePageRequest(r, defaultHistoryPageSize, maxHistoryPageSize)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	entries, total, err := h.tasks.GetHistory(r.Context(), actorID, taskID, page.Offset(), page.PageSize)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	resp := make([]dto.HistoryEntryResponse, 0, len(entries))
	for _, e := range entries {
		resp = append(resp, dto.NewHistoryEntryResponse(e))
	}
	httpx.WriteList(w, http.StatusOK, resp, httpx.NewMeta(page, total))
}

func parseListFilter(r *http.Request) (model.ListFilter, error) {
	q := r.URL.Query()

	teamIDStr := q.Get("team_id")
	if teamIDStr == "" {
		return model.ListFilter{}, apperr.Validation("team_id", "is required")
	}
	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		return model.ListFilter{}, apperr.Validation("team_id", "must be a positive integer")
	}

	filter := model.ListFilter{TeamID: teamID}

	if s := q.Get("status"); s != "" {
		status := model.Status(s)
		if !status.Valid() {
			return model.ListFilter{}, apperr.Validation("status", "must be one of todo, in_progress, done")
		}
		filter.Status = &status
	}

	if a := q.Get("assignee_to"); a != "" {
		assigneeID, err := strconv.ParseInt(a, 10, 64)
		if err != nil || assigneeID <= 0 {
			return model.ListFilter{}, apperr.Validation("assignee_to", "must be a positive integer")
		}
		filter.AssigneeTo = &assigneeID
	}

	return filter, nil
}

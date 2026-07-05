package handler

import (
	"net/http"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
	"github.com/zhuk/team-task-service/internal/service"
)

type TeamHandler struct {
	teams *service.TeamService
}

func NewTeamHandler(teams *service.TeamService) *TeamHandler {
	return &TeamHandler{teams: teams}
}

// Create godoc
// @Summary      Create a team (caller becomes owner)
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateTeamRequest true "Team"
// @Success      201 {object} httpx.Envelope{data=dto.TeamResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      401 {object} httpx.ErrorEnvelope
// @Router       /teams [post]
func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req dto.CreateTeamRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	team, err := h.teams.Create(r.Context(), actorID, req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteData(w, http.StatusCreated, dto.NewTeamResponse(team))
}

// List godoc
// @Summary      List teams the caller belongs to
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} httpx.Envelope{data=[]dto.TeamMembershipResponse}
// @Failure      401 {object} httpx.ErrorEnvelope
// @Router       /teams [get]
func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	memberships, err := h.teams.ListForUser(r.Context(), actorID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	resp := make([]dto.TeamMembershipResponse, 0, len(memberships))
	for _, m := range memberships {
		resp = append(resp, dto.NewTeamMembershipResponse(m))
	}
	httpx.WriteData(w, http.StatusOK, resp)
}

// Invite godoc
// @Summary      Invite an existing user to a team
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Team ID"
// @Param        request body dto.InviteRequest true "Invitee email"
// @Success      201 {object} httpx.Envelope{data=dto.InviteResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      403 {object} httpx.ErrorEnvelope
// @Failure      404 {object} httpx.ErrorEnvelope
// @Failure      409 {object} httpx.ErrorEnvelope
// @Router       /teams/{id}/invite [post]
func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	actorID, err := requireUserID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	teamID, err := parseIDParam(r, "id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	var req dto.InviteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	invitee, err := h.teams.Invite(r.Context(), actorID, teamID, req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteData(w, http.StatusCreated, dto.InviteResponse{
		TeamID: teamID,
		UserID: invitee.ID,
		Role:   model.RoleMember,
	})
}

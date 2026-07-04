package dto

import (
	"strings"
	"time"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
)

type CreateTeamRequest struct {
	Name string `json:"name"`
}

func (r CreateTeamRequest) Validate() *apperr.Error {
	if strings.TrimSpace(r.Name) == "" || len(r.Name) > 255 {
		return apperr.Validation("name", "is required and must be at most 255 characters")
	}
	return nil
}

type TeamResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func NewTeamResponse(t *model.Team) TeamResponse {
	return TeamResponse{ID: t.ID, Name: t.Name, CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt}
}

type TeamMembershipResponse struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Role      model.Role `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
}

func NewTeamMembershipResponse(m model.TeamMembership) TeamMembershipResponse {
	return TeamMembershipResponse{ID: m.ID, Name: m.Name, Role: m.Role, CreatedAt: m.CreatedAt}
}

type InviteRequest struct {
	Email string `json:"email"`
}

func (r InviteRequest) Validate() *apperr.Error {
	return validateEmail(r.Email)
}

type InviteResponse struct {
	TeamID int64      `json:"team_id"`
	UserID int64      `json:"user_id"`
	Role   model.Role `json:"role"`
}

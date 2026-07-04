package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/notify"
)

type TeamService struct {
	teams    TeamRepository
	users    UserRepository
	notifier notify.Notifier
}

func NewTeamService(teams TeamRepository, users UserRepository, notifier notify.Notifier) *TeamService {
	return &TeamService{teams: teams, users: users, notifier: notifier}
}

func (s *TeamService) Create(ctx context.Context, actorID int64, req dto.CreateTeamRequest) (*model.Team, error) {
	return s.teams.CreateWithOwner(ctx, req.Name, actorID)
}

func (s *TeamService) ListForUser(ctx context.Context, userID int64) ([]model.TeamMembership, error) {
	return s.teams.ListForUser(ctx, userID)
}

// Invite adds an already-registered user to the team as a member. Sending
// the invite email is best-effort: a Notifier failure is logged but does
// not fail the request, since the membership itself already succeeded.
func (s *TeamService) Invite(ctx context.Context, actorID, teamID int64, req dto.InviteRequest) (*model.User, error) {
	actorRole, err := s.teams.GetRole(ctx, teamID, actorID)
	if err != nil {
		return nil, err
	}
	if !actorRole.CanInvite() {
		return nil, apperr.Forbidden("only owner or admin can invite members")
	}

	invitee, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr.Code == apperr.CodeNotFound {
			return nil, apperr.NotFound("no registered user with this email")
		}
		return nil, err
	}

	if err := s.teams.AddMember(ctx, teamID, invitee.ID, model.RoleMember); err != nil {
		return nil, err
	}

	if err := s.notifier.SendInvite(ctx, invitee.Email, teamID); err != nil {
		slog.Warn("invite email failed (best-effort, membership already persisted)",
			"error", err, "team_id", teamID, "email", invitee.Email)
	}

	return invitee, nil
}

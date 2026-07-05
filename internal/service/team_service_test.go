package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/service"
)

func TestTeamService_Invite_ForbiddenForMember(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleMember)
	users := newFakeUserRepo()
	_, err := users.Create(context.Background(), &model.User{Email: "invitee@example.com", Name: "Bob"})
	require.NoError(t, err)

	svc := service.NewTeamService(teams, users, &fakeNotifier{})
	_, err = svc.Invite(context.Background(), 100, 1, dto.InviteRequest{Email: "invitee@example.com"})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeForbidden, appErr.Code)
}

func TestTeamService_Invite_AllowedForAdmin(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleAdmin)
	users := newFakeUserRepo()
	invitee, err := users.Create(context.Background(), &model.User{Email: "invitee@example.com", Name: "Bob"})
	require.NoError(t, err)

	notifier := &fakeNotifier{}
	svc := service.NewTeamService(teams, users, notifier)

	got, err := svc.Invite(context.Background(), 100, 1, dto.InviteRequest{Email: "invitee@example.com"})
	require.NoError(t, err)
	require.Equal(t, invitee.ID, got.ID)
	require.Equal(t, 1, notifier.calls)
}

func TestTeamService_Invite_UnknownEmailNotFound(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleOwner)
	users := newFakeUserRepo()

	svc := service.NewTeamService(teams, users, &fakeNotifier{})
	_, err := svc.Invite(context.Background(), 100, 1, dto.InviteRequest{Email: "nobody@example.com"})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeNotFound, appErr.Code)
}

func TestTeamService_Invite_AlreadyMemberConflict(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleOwner)
	users := newFakeUserRepo()
	invitee, err := users.Create(context.Background(), &model.User{Email: "invitee@example.com", Name: "Bob"})
	require.NoError(t, err)
	require.NoError(t, teams.AddMember(context.Background(), 1, invitee.ID, model.RoleMember))

	svc := service.NewTeamService(teams, users, &fakeNotifier{})
	_, err = svc.Invite(context.Background(), 100, 1, dto.InviteRequest{Email: "invitee@example.com"})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeConflict, appErr.Code)
}

func TestTeamService_Invite_NotifierFailureDoesNotFailRequest(t *testing.T) {
	teams := newFakeTeamRepo()
	teams.setRole(1, 100, model.RoleOwner)
	users := newFakeUserRepo()
	invitee, err := users.Create(context.Background(), &model.User{Email: "invitee@example.com", Name: "Bob"})
	require.NoError(t, err)

	notifier := &fakeNotifier{err: errors.New("smtp down")}
	svc := service.NewTeamService(teams, users, notifier)

	got, err := svc.Invite(context.Background(), 100, 1, dto.InviteRequest{Email: "invitee@example.com"})
	require.NoError(t, err)
	require.Equal(t, invitee.ID, got.ID)
}

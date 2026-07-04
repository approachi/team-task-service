package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/model"
)

func TestRole_CanInvite(t *testing.T) {
	require.True(t, model.RoleOwner.CanInvite())
	require.True(t, model.RoleAdmin.CanInvite())
	require.False(t, model.RoleMember.CanInvite())
}

func TestRole_CanManageAnyTask(t *testing.T) {
	require.True(t, model.RoleOwner.CanManageAnyTask())
	require.True(t, model.RoleAdmin.CanManageAnyTask())
	require.False(t, model.RoleMember.CanManageAnyTask())
}

func TestRole_Valid(t *testing.T) {
	require.True(t, model.RoleOwner.Valid())
	require.True(t, model.RoleAdmin.Valid())
	require.True(t, model.RoleMember.Valid())
	require.False(t, model.Role("superadmin").Valid())
}

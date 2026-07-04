package model

// Role mirrors the team_members.role ENUM('owner','admin','member') directly
// — see docs/COVER_LETTER.md for why authorization always reads the
// database instead of caching the role in the JWT.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// CanInvite reports whether this role may invite new members to the team.
func (r Role) CanInvite() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanManageAnyTask reports whether this role may update or reassign any
// task in the team, not just tasks assigned to itself.
func (r Role) CanManageAnyTask() bool {
	return r == RoleOwner || r == RoleAdmin
}

// Valid reports whether r is one of the known roles. Phase 1 never accepts
// a role as request input (the server always assigns it), but this exists
// for the DTO boundary of any future endpoint that does.
func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

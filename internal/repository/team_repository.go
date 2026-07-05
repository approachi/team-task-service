package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
)

const teamColumns = "id, name, created_by, created_at, updated_at"

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// CreateWithOwner inserts the team and its owner membership row in one
// transaction so a team never exists without an owner.
func (r *TeamRepository) CreateWithOwner(ctx context.Context, name string, creatorID int64) (*model.Team, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `INSERT INTO teams (name, created_by) VALUES (?, ?)`, name, creatorID)
	if err != nil {
		return nil, fmt.Errorf("insert team: %w", err)
	}
	teamID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get inserted team id: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, creatorID, model.RoleOwner,
	); err != nil {
		return nil, fmt.Errorf("insert owner membership: %w", err)
	}

	var team model.Team
	query := fmt.Sprintf(`SELECT %s FROM teams WHERE id = ?`, teamColumns)
	if err := tx.GetContext(ctx, &team, query, teamID); err != nil {
		return nil, fmt.Errorf("get created team: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &team, nil
}

func (r *TeamRepository) ListForUser(ctx context.Context, userID int64) ([]model.TeamMembership, error) {
	memberships := make([]model.TeamMembership, 0)
	query := `
		SELECT t.id, t.name, tm.role, t.created_at
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC`
	if err := r.db.SelectContext(ctx, &memberships, query, userID); err != nil {
		return nil, fmt.Errorf("list teams for user: %w", err)
	}
	return memberships, nil
}

// GetRole implements the (structural) service.TeamAuthorizer interface
// consumed by the task service to authorize task operations.
func (r *TeamRepository) GetRole(ctx context.Context, teamID, userID int64) (model.Role, error) {
	var role model.Role
	err := r.db.GetContext(ctx,
		&role,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", apperr.Forbidden("not a member of this team")
		}
		return "", fmt.Errorf("get team role: %w", err)
	}
	return role, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID int64, role model.Role) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, userID, role,
	)
	if err != nil {
		if isDuplicateKeyErr(err) {
			return apperr.Conflict("user is already a member of this team")
		}
		return fmt.Errorf("insert team member: %w", err)
	}
	return nil
}

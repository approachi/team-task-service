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

const userColumns = "id, email, password_hash, name, created_at, updated_at"

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) (*model.User, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		u.Email, u.PasswordHash, u.Name,
	)
	if err != nil {
		if isDuplicateKeyErr(err) {
			return nil, apperr.Conflict("email is already registered")
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get inserted user id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	query := fmt.Sprintf(`SELECT %s FROM users WHERE id = ?`, userColumns)
	if err := r.db.GetContext(ctx, &u, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	query := fmt.Sprintf(`SELECT %s FROM users WHERE email = ?`, userColumns)
	if err := r.db.GetContext(ctx, &u, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("user not found")
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

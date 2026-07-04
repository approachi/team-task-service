package model

import "time"

type Team struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedBy int64     `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// TeamMembership is the shape returned by "teams I belong to" queries: a
// team joined with the caller's own role in it.
type TeamMembership struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Role      Role      `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

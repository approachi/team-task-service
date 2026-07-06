// Package testhelper spins up a real MySQL via testcontainers-go and
// applies goose migrations against it, so integration tests exercise the
// same schema/constraints as production rather than mocks.
package testhelper

import (
	"context"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

// StartMySQL starts a MySQL container, applies all goose migrations found
// in migrationsDir, and returns a connected *sqlx.DB. The container and the
// DB connection are torn down automatically via t.Cleanup.
func StartMySQL(t *testing.T, migrationsDir string) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("team_task_test"),
		mysql.WithUsername("app"),
		mysql.WithPassword("app_password"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminate mysql container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		t.Fatalf("get mysql connection string: %v", err)
	}

	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		t.Fatalf("connect to mysql: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := goose.SetDialect("mysql"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(db.DB, migrationsDir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

// Package db opens the shared *sqlx.DB connection pool used by every
// repository in the service.
//
// TODO: нет реплики на чтение (все репозитории, включая тяжёлые отчёты в
// analytics_repository.go, ходят в primary) и нет явного query-level
// timeout поверх ctx на медленных аналитических запросах — один тяжёлый
// отчёт может держать соединение из общего пула дольше, чем стоило бы.
package db

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"

	"github.com/zhuk/team-task-service/internal/config"
)

func Connect(cfg config.DBConfig) (*sqlx.DB, error) {
	conn, err := sqlx.Connect("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect to mysql: %w", err)
	}

	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime.Duration))

	return conn, nil
}

package sqlc

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/juho05/crossonic-server"
	"github.com/juho05/log"
	migrate "github.com/rubenv/sql-migrate"
)

func Connect(dsn string) (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("connect DB: %w", err)
	}
	return conn, nil
}

func Close(conn *pgxpool.Pool) error {
	conn.Close()
	return nil
}

func AutoMigrate(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	defer db.Close()
	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: http.FS(crossonic.MigrationsFS),
	}
	log.Trace("Migrating database...")
	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	log.Tracef("Applied %d migrations!", n)
	return nil
}

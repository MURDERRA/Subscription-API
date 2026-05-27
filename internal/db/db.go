package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	"subscription-service/migrations"
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	slog.Info("подключение к БД установлено")
	return db, nil
}

func Migrate(dsn string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("iofs.New: %w", err)
	}

	// golang-migrate ожидает postgres:// DSN
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate.Up: %w", err)
	}
	slog.Info("миграции применены")
	return nil
}

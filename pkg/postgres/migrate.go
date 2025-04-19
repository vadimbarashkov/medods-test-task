package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var poolParamPattern = regexp.MustCompile(`(^|&)pool_[^=]+=[^&]*`)

func removePoolParams(dsn string) string {
	parts := strings.SplitN(dsn, "?", 2)
	if len(parts) != 2 {
		return dsn
	}

	base := parts[0]
	query := parts[1]

	cleanQuery := poolParamPattern.ReplaceAllString(query, "")
	cleanQuery = strings.Trim(cleanQuery, "&")

	if cleanQuery == "" {
		return base
	}

	return base + "?" + cleanQuery
}

func ApplyMigrations(ctx context.Context, dsn, path string) error {
	dsn = removePoolParams(dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migrate driver: %w", err)
	}

	if !strings.HasPrefix(path, "file://") {
		path = "file://" + path
	}

	m, err := migrate.NewWithDatabaseInstance(path, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instanse: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

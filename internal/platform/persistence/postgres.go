package persistence

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrPostgresDSNRequired = errors.New("postgres dsn is required")

//go:embed schema.sql
var postgresSchema string

type postgresOpenFunc func(driverName string, dsn string) (*sql.DB, error)
type postgresPingFunc func(ctx context.Context, db *sql.DB) error
type postgresMigrateFunc func(ctx context.Context, db *sql.DB) error

type postgresOptions struct {
	driverName string
	open       postgresOpenFunc
	ping       postgresPingFunc
	migrate    postgresMigrateFunc
}

type PostgresOption func(*postgresOptions)

func OpenPostgres(ctx context.Context, dsn string, options ...PostgresOption) (*sql.DB, error) {
	trimmedDSN := strings.TrimSpace(dsn)
	if trimmedDSN == "" {
		return nil, ErrPostgresDSNRequired
	}

	resolvedOptions := postgresOptions{
		driverName: "pgx",
		open:       sql.Open,
		ping: func(ctx context.Context, db *sql.DB) error {
			return db.PingContext(ctx)
		},
		migrate: RunPostgresSchema,
	}
	for _, option := range options {
		option(&resolvedOptions)
	}

	db, err := resolvedOptions.open(resolvedOptions.driverName, trimmedDSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := resolvedOptions.ping(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := resolvedOptions.migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run postgres schema: %w", err)
	}

	return db, nil
}

func RunPostgresSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range splitSQLStatements(postgresSchema) {
		if strings.TrimSpace(statement) == "" {
			continue
		}

		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("exec schema statement: %w", err)
		}
	}

	return nil
}

func splitSQLStatements(schema string) []string {
	statements := strings.Split(schema, ";")
	result := make([]string, 0, len(statements))
	for _, statement := range statements {
		trimmed := strings.TrimSpace(statement)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func withPostgresOpen(open postgresOpenFunc) PostgresOption {
	return func(options *postgresOptions) {
		options.open = open
	}
}

func withPostgresPing(ping postgresPingFunc) PostgresOption {
	return func(options *postgresOptions) {
		options.ping = ping
	}
}

func withPostgresMigrate(migrate postgresMigrateFunc) PostgresOption {
	return func(options *postgresOptions) {
		options.migrate = migrate
	}
}

package persistence

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
)

func TestOpenPostgresReturnsErrorWhenDSNMissing(t *testing.T) {
	db, err := OpenPostgres(context.Background(), "")
	if err == nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatalf("expected missing dsn to return error")
	}
}

func TestOpenPostgresPingsAndMigratesDatabase(t *testing.T) {
	calls := make([]string, 0, 3)

	db, err := OpenPostgres(
		context.Background(),
		"postgres://postgres:postgres@localhost:5432/supportpilot?sslmode=disable",
		withPostgresOpen(func(driverName string, dsn string) (*sql.DB, error) {
			calls = append(calls, "open")
			return &sql.DB{}, nil
		}),
		withPostgresPing(func(ctx context.Context, db *sql.DB) error {
			calls = append(calls, "ping")
			return nil
		}),
		withPostgresMigrate(func(ctx context.Context, db *sql.DB) error {
			calls = append(calls, "migrate")
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("expected open postgres success, got error: %v", err)
	}
	if db == nil {
		t.Fatalf("expected database handle to be returned")
	}

	if !reflect.DeepEqual(calls, []string{"open", "ping", "migrate"}) {
		t.Fatalf("expected postgres bootstrap order open->ping->migrate, got %v", calls)
	}
}

package database

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSQLiteConnectionEnforcesConstraintsAndTransactions(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, Config{Driver: SQLite, SQLitePath: filepath.Join(t.TempDir(), "nested", "stream.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(ctx, `CREATE TABLE parents (id INTEGER PRIMARY KEY); CREATE TABLE children (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parents(id), created_at TIMESTAMP);`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO children (parent_id) VALUES ($1)`, 99); err == nil {
		t.Fatal("foreign key constraint was not enforced")
	}
	if _, err := db.Exec(ctx, `INSERT INTO parents (id) VALUES ($1)`, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO parents (id) VALUES ($1)`, 1); !IsUniqueViolation(err) {
		t.Fatalf("duplicate key was not classified: %v", err)
	}
	tx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 8, 12, 34, 56, 0, time.UTC)
	// The same parameter must bind consistently even when first encountered
	// out of order, as it does in the existing repository queries.
	var id int64
	var stored time.Time
	if err := tx.QueryRow(ctx, `INSERT INTO children (created_at, parent_id) VALUES ($2, $1) RETURNING id, created_at`, 1, now).Scan(&id, &stored); err != nil {
		t.Fatal(err)
	}
	if !stored.Equal(now) {
		t.Fatalf("timestamp round trip: got %v, want %v", stored, now)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	err = db.QueryRow(ctx, `SELECT id FROM children WHERE id = $1`, id).Scan(&id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rolled-back insert survived: %v", err)
	}
}

func TestOpenRejectsUnsupportedProviderAndRedactsCredentials(t *testing.T) {
	_, err := Open(context.Background(), Config{Driver: "mysql"})
	if err == nil || !strings.Contains(err.Error(), "unsupported DATABASE_DRIVER") {
		t.Fatalf("unexpected driver error: %v", err)
	}
	_, err = Open(context.Background(), Config{Driver: Postgres, URL: "postgres://user:private-password@localhost:invalid-port/database"})
	if err == nil || strings.Contains(err.Error(), "private-password") {
		t.Fatalf("invalid URL error exposed credentials: %v", err)
	}
}

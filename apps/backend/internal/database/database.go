// Package database supplies the connection and transaction boundary shared by
// PostgreSQL and SQLite repositories. SQL dialect differences remain explicit
// in repositories and migrations, rather than translating arbitrary SQL.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"modernc.org/sqlite"
)

type Dialect string

const (
	Postgres Dialect = "postgres"
	SQLite   Dialect = "sqlite"
)

// Executor is implemented by both a connection pool and an open transaction.
// Parameters use $1, $2, etc.; SQLite binds them as numbered ?1, ?2 parameters.
type Executor interface {
	Query(context.Context, string, ...any) (*sql.Rows, error)
	QueryRow(context.Context, string, ...any) *sql.Row
	Exec(context.Context, string, ...any) (sql.Result, error)
	Dialect() Dialect
}

type DB interface {
	Executor
	BeginTx(context.Context) (Tx, error)
	Ping(context.Context) error
	Close() error
}

type Tx interface {
	Executor
	Commit(context.Context) error
	Rollback(context.Context) error
}

type Config struct {
	Driver     Dialect
	URL        string
	SQLitePath string
}

func ConfigFromEnv() Config {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_DRIVER")))
	if driver == "" || driver == "postgresql" {
		driver = string(Postgres)
	}
	if driver == "sqlite3" {
		driver = string(SQLite)
	}
	return Config{
		Driver:     Dialect(driver),
		URL:        envOrDefault("DATABASE_URL", "postgres://ad:ad@localhost:5432/ad_service?sslmode=disable"),
		SQLitePath: envOrDefault("DATABASE_SQLITE_PATH", "./data/stream.db"),
	}
}

func Open(ctx context.Context, cfg Config) (DB, error) {
	var raw *sql.DB
	var err error
	switch cfg.Driver {
	case Postgres:
		pgConfig, parseErr := pgx.ParseConfig(cfg.URL)
		if parseErr != nil {
			// Driver parse errors can include credentials from the supplied DSN.
			return nil, fmt.Errorf("invalid PostgreSQL connection configuration")
		}
		pgConfig.ConnectTimeout = 5 * time.Second
		pgConfig.RuntimeParams["timezone"] = "UTC"
		raw = stdlib.OpenDB(*pgConfig)
		raw.SetMaxOpenConns(10)
		raw.SetMaxIdleConns(10)
		raw.SetConnMaxLifetime(30 * time.Minute)
	case SQLite:
		if strings.TrimSpace(cfg.SQLitePath) == "" {
			return nil, fmt.Errorf("DATABASE_SQLITE_PATH is required for SQLite")
		}
		var dsn string
		if cfg.SQLitePath == ":memory:" {
			dsn = ":memory:"
		} else {
			absolute, pathErr := filepath.Abs(cfg.SQLitePath)
			if pathErr != nil {
				return nil, fmt.Errorf("resolve SQLite database path: %w", pathErr)
			}
			if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
				return nil, fmt.Errorf("create SQLite database directory: %w", err)
			}
			dsn = (&url.URL{Scheme: "file", Path: absolute}).String()
		}
		// Every connection enforces FKs. BEGIN IMMEDIATE serializes credit writes
		// before they read a balance, preserving PostgreSQL row-lock semantics.
		params := url.Values{
			"_pragma":      {"foreign_keys(1)", "busy_timeout(5000)", "journal_mode(WAL)"},
			"_txlock":      {"immediate"},
			"_time_format": {"sqlite"},
		}
		raw, err = sql.Open("sqlite", dsn+"?"+params.Encode())
		if err != nil {
			return nil, fmt.Errorf("open SQLite database: %w", err)
		}
		// SQLite is a local single-writer provider. One connection also keeps
		// :memory: databases alive and avoids contention within this process.
		raw.SetMaxOpenConns(1)
		raw.SetMaxIdleConns(1)
	default:
		return nil, fmt.Errorf("unsupported DATABASE_DRIVER %q; use postgres or sqlite", cfg.Driver)
	}
	connection := &connection{raw: raw, dialect: cfg.Driver}
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := connection.Ping(connectCtx); err != nil {
		_ = raw.Close()
		// Connection errors may contain credentials or other DSN parameters.
		return nil, fmt.Errorf("connect to %s database: check connection settings and availability", cfg.Driver)
	}
	return connection, nil
}

type connection struct {
	raw     *sql.DB
	dialect Dialect
}

func (d *connection) Dialect() Dialect               { return d.dialect }
func (d *connection) Ping(ctx context.Context) error { return d.raw.PingContext(ctx) }
func (d *connection) Close() error                   { return d.raw.Close() }
func (d *connection) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.raw.QueryContext(ctx, bind(d.dialect, query), arguments(d.dialect, args)...)
}
func (d *connection) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.raw.QueryRowContext(ctx, bind(d.dialect, query), arguments(d.dialect, args)...)
}
func (d *connection) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.raw.ExecContext(ctx, bind(d.dialect, query), arguments(d.dialect, args)...)
}
func (d *connection) BeginTx(ctx context.Context) (Tx, error) {
	transaction, err := d.raw.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &transactionConnection{raw: transaction, dialect: d.dialect}, nil
}

type transactionConnection struct {
	raw     *sql.Tx
	dialect Dialect
}

func (t *transactionConnection) Dialect() Dialect { return t.dialect }
func (t *transactionConnection) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.raw.QueryContext(ctx, bind(t.dialect, query), arguments(t.dialect, args)...)
}
func (t *transactionConnection) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return t.raw.QueryRowContext(ctx, bind(t.dialect, query), arguments(t.dialect, args)...)
}
func (t *transactionConnection) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.raw.ExecContext(ctx, bind(t.dialect, query), arguments(t.dialect, args)...)
}
func (t *transactionConnection) Commit(_ context.Context) error   { return t.raw.Commit() }
func (t *transactionConnection) Rollback(_ context.Context) error { return t.raw.Rollback() }

// ForUpdate is used only on fixed repository queries inside a transaction.
// SQLite's immediate write transaction supplies the equivalent writer lock.
func ForUpdate(db Executor) string {
	if db.Dialect() == Postgres {
		return " FOR UPDATE"
	}
	return ""
}

// IsUniqueViolation normalizes the two providers' duplicate-key errors.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && (sqliteErr.Code() == 2067 || sqliteErr.Code() == 1555)
}

var parameter = regexp.MustCompile(`\$(\d+)`)

func bind(dialect Dialect, query string) string {
	if dialect == SQLite {
		return parameter.ReplaceAllString(query, `?$1`)
	}
	return query
}

func arguments(dialect Dialect, args []any) []any {
	if dialect != SQLite {
		return args
	}
	values := make([]any, len(args))
	for i, arg := range args {
		// Fixed-width UTC timestamps preserve chronological order for SQLite
		// comparisons and the driver's TIMESTAMP scanner restores time.Time.
		switch value := arg.(type) {
		case time.Time:
			values[i] = value.UTC().Format("2006-01-02 15:04:05.000000000+00:00")
		case *time.Time:
			if value != nil {
				values[i] = value.UTC().Format("2006-01-02 15:04:05.000000000+00:00")
			}
		default:
			values[i] = arg
		}
	}
	return values
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

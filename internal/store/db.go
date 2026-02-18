package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const createSchema = `
CREATE TABLE IF NOT EXISTS vault_fields (
	id          TEXT PRIMARY KEY,
	category    TEXT NOT NULL,
	field_name  TEXT NOT NULL,
	value       TEXT NOT NULL,
	sensitivity TEXT NOT NULL DEFAULT 'standard',
	updated_at  TEXT NOT NULL,
	version     INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS vault_access_log (
	id         TEXT PRIMARY KEY,
	consumer   TEXT NOT NULL,
	scope      TEXT NOT NULL,
	action     TEXT NOT NULL,
	purpose    TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS vault_tokens (
	token      TEXT PRIMARY KEY,
	consumer   TEXT NOT NULL,
	scope      TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	usage      TEXT NOT NULL DEFAULT 'multi',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS vault_meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_fields_category ON vault_fields(category);
CREATE INDEX IF NOT EXISTS idx_fields_sensitivity ON vault_fields(sensitivity);
CREATE INDEX IF NOT EXISTS idx_tokens_expires ON vault_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_access_log_created ON vault_access_log(created_at);
`

// DB wraps a *sql.DB with vault-specific operations.
type DB struct {
	conn *sql.DB
}

// Open opens or creates the vault database at the given path.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	} {
		if _, err := conn.Exec(pragma); err != nil {
			conn.Close()
			return nil, fmt.Errorf("setting %s: %w", pragma, err)
		}
	}

	if _, err := conn.Exec(createSchema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

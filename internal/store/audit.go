package store

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// AuditEntry represents a row in vault_access_log.
type AuditEntry struct {
	ID        string
	Consumer  string
	Scope     string
	Action    string
	Purpose   string
	CreatedAt time.Time
}

// LogAccess writes an audit entry.
func (d *DB) LogAccess(entry AuditEntry) error {
	if entry.ID == "" {
		b := make([]byte, 16)
		rand.Read(b)
		entry.ID = hex.EncodeToString(b)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	_, err := d.conn.Exec(
		`INSERT INTO vault_access_log (id, consumer, scope, action, purpose, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Consumer, entry.Scope, entry.Action, entry.Purpose,
		entry.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetAuditLog retrieves recent audit entries, newest first.
func (d *DB) GetAuditLog(limit int) ([]AuditEntry, error) {
	rows, err := d.conn.Query(
		"SELECT id, consumer, scope, action, purpose, created_at FROM vault_access_log ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var createdAt string
		if err := rows.Scan(&e.ID, &e.Consumer, &e.Scope, &e.Action, &e.Purpose, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

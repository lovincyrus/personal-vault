package store

import (
	"database/sql"
	"time"
)

// Field represents a row in vault_fields.
type Field struct {
	ID          string
	Category    string
	FieldName   string
	Value       string // encrypted ciphertext (base64)
	Sensitivity string
	UpdatedAt   time.Time
	Version     int
}

// SetField upserts a field. If the field exists, bumps version.
func (d *DB) SetField(f Field) error {
	_, err := d.conn.Exec(
		`INSERT INTO vault_fields (id, category, field_name, value, sensitivity, updated_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, 1)
		 ON CONFLICT(id) DO UPDATE SET
			value = excluded.value,
			sensitivity = CASE WHEN excluded.sensitivity != '' THEN excluded.sensitivity ELSE vault_fields.sensitivity END,
			updated_at = excluded.updated_at,
			version = vault_fields.version + 1`,
		f.ID, f.Category, f.FieldName, f.Value, f.Sensitivity, f.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetField retrieves a single field by ID (includes encrypted value).
func (d *DB) GetField(id string) (*Field, error) {
	var f Field
	var updatedAt string
	err := d.conn.QueryRow(
		"SELECT id, category, field_name, value, sensitivity, updated_at, version FROM vault_fields WHERE id = ?",
		id,
	).Scan(&f.ID, &f.Category, &f.FieldName, &f.Value, &f.Sensitivity, &updatedAt, &f.Version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &f, nil
}

// ListFields returns all field metadata (no values).
func (d *DB) ListFields() ([]Field, error) {
	rows, err := d.conn.Query(
		"SELECT id, category, field_name, sensitivity, updated_at, version FROM vault_fields ORDER BY category, field_name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []Field
	for rows.Next() {
		var f Field
		var updatedAt string
		if err := rows.Scan(&f.ID, &f.Category, &f.FieldName, &f.Sensitivity, &updatedAt, &f.Version); err != nil {
			return nil, err
		}
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

// ListFieldsByCategory returns field metadata for a category (no values).
func (d *DB) ListFieldsByCategory(category string) ([]Field, error) {
	rows, err := d.conn.Query(
		"SELECT id, category, field_name, sensitivity, updated_at, version FROM vault_fields WHERE category = ? ORDER BY field_name",
		category,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []Field
	for rows.Next() {
		var f Field
		var updatedAt string
		if err := rows.Scan(&f.ID, &f.Category, &f.FieldName, &f.Sensitivity, &updatedAt, &f.Version); err != nil {
			return nil, err
		}
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

// GetFieldsByCategory returns fields in a category (with encrypted values).
func (d *DB) GetFieldsByCategory(category string) ([]Field, error) {
	rows, err := d.conn.Query(
		"SELECT id, category, field_name, value, sensitivity, updated_at, version FROM vault_fields WHERE category = ? ORDER BY field_name",
		category,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []Field
	for rows.Next() {
		var f Field
		var updatedAt string
		if err := rows.Scan(&f.ID, &f.Category, &f.FieldName, &f.Value, &f.Sensitivity, &updatedAt, &f.Version); err != nil {
			return nil, err
		}
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

// GetAllFields returns all fields including encrypted values.
func (d *DB) GetAllFields() ([]Field, error) {
	rows, err := d.conn.Query(
		"SELECT id, category, field_name, value, sensitivity, updated_at, version FROM vault_fields ORDER BY category, field_name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []Field
	for rows.Next() {
		var f Field
		var updatedAt string
		if err := rows.Scan(&f.ID, &f.Category, &f.FieldName, &f.Value, &f.Sensitivity, &updatedAt, &f.Version); err != nil {
			return nil, err
		}
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

// DeleteField removes a field by ID.
func (d *DB) DeleteField(id string) error {
	_, err := d.conn.Exec("DELETE FROM vault_fields WHERE id = ?", id)
	return err
}

// SetSensitivity updates the sensitivity tier of a field.
func (d *DB) SetSensitivity(id, tier string) error {
	_, err := d.conn.Exec(
		"UPDATE vault_fields SET sensitivity = ?, updated_at = ? WHERE id = ?",
		tier, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

// FieldCount returns total number of fields.
func (d *DB) FieldCount() (int, error) {
	var count int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM vault_fields").Scan(&count)
	return count, err
}

// CategoryCounts returns a map of category -> field count.
func (d *DB) CategoryCounts() (map[string]int, error) {
	rows, err := d.conn.Query("SELECT category, COUNT(*) FROM vault_fields GROUP BY category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var cat string
		var count int
		if err := rows.Scan(&cat, &count); err != nil {
			return nil, err
		}
		counts[cat] = count
	}
	return counts, rows.Err()
}

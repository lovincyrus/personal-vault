package store

import "database/sql"

// SetMeta upserts a key-value pair in vault_meta.
func (d *DB) SetMeta(key, value string) error {
	_, err := d.conn.Exec(
		`INSERT INTO vault_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

// GetMeta retrieves a value by key. Returns empty string if not found.
func (d *DB) GetMeta(key string) (string, error) {
	var value string
	err := d.conn.QueryRow("SELECT value FROM vault_meta WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// IsInitialized checks if the vault has been initialized.
func (d *DB) IsInitialized() (bool, error) {
	salt, err := d.GetMeta("salt")
	if err != nil {
		return false, err
	}
	return salt != "", nil
}

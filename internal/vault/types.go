package vault

import "time"

// VaultStatus describes the current state of the vault.
type VaultStatus struct {
	Initialized bool           `json:"initialized"`
	Locked      bool           `json:"locked"`
	FieldCount  int            `json:"field_count"`
	Categories  map[string]int `json:"categories"`
}

// FieldInfo is a decrypted field returned to callers.
type FieldInfo struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	FieldName   string    `json:"field_name"`
	Value       string    `json:"value,omitempty"`
	Sensitivity string    `json:"sensitivity"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
}

// ContextBundle is a full decrypted dump grouped by category.
type ContextBundle struct {
	Categories map[string][]FieldInfo `json:"categories"`
}

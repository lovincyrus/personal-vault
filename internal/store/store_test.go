package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tmpDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpen_CreatesSchema(t *testing.T) {
	db := tmpDB(t)
	// Verify all tables exist by querying them
	for _, table := range []string{"vault_fields", "vault_access_log", "vault_tokens", "vault_meta"} {
		var name string
		err := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s not found: %v", table, err)
		}
	}
}

func TestOpen_WALMode(t *testing.T) {
	db := tmpDB(t)
	var mode string
	db.conn.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if mode != "wal" {
		t.Fatalf("expected WAL mode, got %s", mode)
	}
}

func TestSetMeta_GetMeta(t *testing.T) {
	db := tmpDB(t)
	if err := db.SetMeta("key1", "value1"); err != nil {
		t.Fatal(err)
	}
	val, err := db.GetMeta("key1")
	if err != nil {
		t.Fatal(err)
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}
}

func TestGetMeta_NotFound(t *testing.T) {
	db := tmpDB(t)
	val, err := db.GetMeta("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Fatalf("expected empty string, got %s", val)
	}
}

func TestIsInitialized(t *testing.T) {
	db := tmpDB(t)
	init, _ := db.IsInitialized()
	if init {
		t.Fatal("should not be initialized")
	}

	db.SetMeta("salt", "some-salt")
	init, _ = db.IsInitialized()
	if !init {
		t.Fatal("should be initialized after setting salt")
	}
}

func TestSetField_Insert(t *testing.T) {
	db := tmpDB(t)
	err := db.SetField(Field{
		ID: "identity.full_name", Category: "identity", FieldName: "full_name",
		Value: "encrypted", Sensitivity: "standard", UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	f, err := db.GetField("identity.full_name")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("field not found")
	}
	if f.Value != "encrypted" {
		t.Fatalf("expected 'encrypted', got %s", f.Value)
	}
	if f.Version != 1 {
		t.Fatalf("expected version 1, got %d", f.Version)
	}
}

func TestSetField_Update_BumpsVersion(t *testing.T) {
	db := tmpDB(t)
	f := Field{
		ID: "identity.dob", Category: "identity", FieldName: "dob",
		Value: "v1", Sensitivity: "sensitive", UpdatedAt: time.Now(),
	}
	db.SetField(f)

	f.Value = "v2"
	f.UpdatedAt = time.Now()
	db.SetField(f)

	got, _ := db.GetField("identity.dob")
	if got.Version != 2 {
		t.Fatalf("expected version 2, got %d", got.Version)
	}
	if got.Value != "v2" {
		t.Fatalf("expected v2, got %s", got.Value)
	}
}

func TestGetField_NotFound(t *testing.T) {
	db := tmpDB(t)
	f, err := db.GetField("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Fatal("expected nil for nonexistent field")
	}
}

func TestListFields_OmitsValues(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "identity.name", Category: "identity", FieldName: "name", Value: "secret", UpdatedAt: time.Now()})

	fields, err := db.ListFields()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Value != "" {
		t.Fatal("ListFields should not include values")
	}
}

func TestListFieldsByCategory_Filters(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "identity.name", Category: "identity", FieldName: "name", Value: "enc1", UpdatedAt: time.Now()})
	db.SetField(Field{ID: "financial.income", Category: "financial", FieldName: "income", Value: "enc2", UpdatedAt: time.Now()})

	fields, _ := db.ListFieldsByCategory("identity")
	if len(fields) != 1 {
		t.Fatalf("expected 1 identity field, got %d", len(fields))
	}
	if fields[0].ID != "identity.name" {
		t.Fatalf("expected identity.name, got %s", fields[0].ID)
	}
}

func TestDeleteField(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "identity.name", Category: "identity", FieldName: "name", Value: "enc", UpdatedAt: time.Now()})
	db.DeleteField("identity.name")

	f, _ := db.GetField("identity.name")
	if f != nil {
		t.Fatal("field should be deleted")
	}
}

func TestSetSensitivity(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "identity.ssn", Category: "identity", FieldName: "ssn", Value: "enc", Sensitivity: "standard", UpdatedAt: time.Now()})
	db.SetSensitivity("identity.ssn", "critical")

	f, _ := db.GetField("identity.ssn")
	if f.Sensitivity != "critical" {
		t.Fatalf("expected critical, got %s", f.Sensitivity)
	}
}

func TestFieldCount(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "a.1", Category: "a", FieldName: "1", Value: "v", UpdatedAt: time.Now()})
	db.SetField(Field{ID: "b.2", Category: "b", FieldName: "2", Value: "v", UpdatedAt: time.Now()})

	count, _ := db.FieldCount()
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestCategoryCounts(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "a.1", Category: "a", FieldName: "1", Value: "v", UpdatedAt: time.Now()})
	db.SetField(Field{ID: "a.2", Category: "a", FieldName: "2", Value: "v", UpdatedAt: time.Now()})
	db.SetField(Field{ID: "b.1", Category: "b", FieldName: "1", Value: "v", UpdatedAt: time.Now()})

	counts, _ := db.CategoryCounts()
	if counts["a"] != 2 || counts["b"] != 1 {
		t.Fatalf("expected a=2 b=1, got %v", counts)
	}
}

func TestCreateToken_GetToken(t *testing.T) {
	db := tmpDB(t)
	tok := Token{
		TokenStr: "abc123", Consumer: "cli", Scope: `["*"]`,
		ExpiresAt: time.Now().Add(time.Hour), Usage: "multi", CreatedAt: time.Now(),
	}
	if err := db.CreateToken(tok); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetToken("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("token not found")
	}
	if got.Consumer != "cli" {
		t.Fatalf("expected cli, got %s", got.Consumer)
	}
}

func TestGetToken_Expired(t *testing.T) {
	db := tmpDB(t)
	tok := Token{
		TokenStr: "expired", Consumer: "cli", Scope: `["*"]`,
		ExpiresAt: time.Now().Add(-time.Hour), Usage: "multi", CreatedAt: time.Now(),
	}
	db.CreateToken(tok)

	got, _ := db.GetToken("expired")
	if got != nil {
		t.Fatal("expired token should return nil")
	}
}

func TestDeleteAllTokens(t *testing.T) {
	db := tmpDB(t)
	now := time.Now()
	db.CreateToken(Token{TokenStr: "t1", Consumer: "c", Scope: "[]", ExpiresAt: now.Add(time.Hour), CreatedAt: now})
	db.CreateToken(Token{TokenStr: "t2", Consumer: "c", Scope: "[]", ExpiresAt: now.Add(time.Hour), CreatedAt: now})

	n, _ := db.DeleteAllTokens()
	if n != 2 {
		t.Fatalf("expected 2 deleted, got %d", n)
	}
}

func TestLogAccess_GetAuditLog(t *testing.T) {
	db := tmpDB(t)
	db.LogAccess(AuditEntry{Consumer: "cli", Scope: "identity.*", Action: "read", Purpose: "test"})
	db.LogAccess(AuditEntry{Consumer: "cli", Scope: "financial.*", Action: "write"})

	entries, err := db.GetAuditLog(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Newest first
	if entries[0].Scope != "financial.*" {
		t.Fatalf("expected financial.* first (newest), got %s", entries[0].Scope)
	}
}

func TestGetFieldsByCategory(t *testing.T) {
	db := tmpDB(t)
	db.SetField(Field{ID: "identity.name", Category: "identity", FieldName: "name", Value: "enc_name", UpdatedAt: time.Now()})
	db.SetField(Field{ID: "identity.dob", Category: "identity", FieldName: "dob", Value: "enc_dob", UpdatedAt: time.Now()})

	fields, err := db.GetFieldsByCategory("identity")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	// Should include values
	if fields[0].Value == "" {
		t.Fatal("GetFieldsByCategory should include values")
	}
}

// Ensure temp dir cleanup works
func TestCleanup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, _ := Open(path)
	db.Close()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("db should exist before cleanup")
	}
}

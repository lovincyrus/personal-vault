package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testPassword = "test-password-123"

func tmpVault(t *testing.T) (*Vault, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".pvault")
	sk, err := Init(dir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	v, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })

	token, err := v.Unlock(testPassword, sk)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	return v, sk
}

func TestInit_CreatesFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".pvault")
	sk, err := Init(dir, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if sk == "" {
		t.Fatal("expected non-empty secret key")
	}

	// Check files exist
	if _, err := os.Stat(filepath.Join(dir, "vault.db")); err != nil {
		t.Fatalf("vault.db not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "secret.key")); err != nil {
		t.Fatalf("secret.key not created: %v", err)
	}
}

func TestInit_AlreadyInitialized(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".pvault")
	Init(dir, testPassword)
	_, err := Init(dir, testPassword)
	if err != ErrAlreadyInit {
		t.Fatalf("expected ErrAlreadyInit, got %v", err)
	}
}

func TestUnlock_WrongPassword(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".pvault")
	sk, _ := Init(dir, testPassword)

	v, _ := Open(dir)
	defer v.Close()

	_, err := v.Unlock("wrong-password", sk)
	if err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
}

func TestUnlock_WrongSecretKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".pvault")
	Init(dir, testPassword)

	v, _ := Open(dir)
	defer v.Close()

	_, err := v.Unlock(testPassword, "deadbeefdeadbeefdeadbeefdeadbeef")
	if err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
}

func TestUnlock_AlreadyUnlocked(t *testing.T) {
	v, _ := tmpVault(t)

	_, err := v.Unlock(testPassword, "doesntmatter")
	if err != ErrAlreadyUnlocked {
		t.Fatalf("expected ErrAlreadyUnlocked, got %v", err)
	}
}

func TestLock(t *testing.T) {
	v, _ := tmpVault(t)
	v.Lock()

	_, err := v.Get("anything")
	if err != ErrLocked {
		t.Fatalf("expected ErrLocked after lock, got %v", err)
	}
}

func TestSetGet_Roundtrip(t *testing.T) {
	v, _ := tmpVault(t)

	if err := v.Set("identity.full_name", "Jane Smith", ""); err != nil {
		t.Fatal(err)
	}

	f, err := v.Get("identity.full_name")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("field not found")
	}
	if f.Value != "Jane Smith" {
		t.Fatalf("expected 'Jane Smith', got %q", f.Value)
	}
	if f.Category != "identity" {
		t.Fatalf("expected category 'identity', got %q", f.Category)
	}
}

func TestSet_InvalidID(t *testing.T) {
	v, _ := tmpVault(t)
	err := v.Set("noperiod", "value", "")
	if err == nil {
		t.Fatal("expected error for invalid field ID")
	}
}

func TestGet_NotFound(t *testing.T) {
	v, _ := tmpVault(t)
	f, err := v.Get("nonexistent.field")
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Fatal("expected nil for nonexistent field")
	}
}

func TestList(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Test", "")
	v.Set("financial.income", "100k", "sensitive")

	fields, err := v.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	// List should not include values
	for _, f := range fields {
		if f.Value != "" {
			t.Fatal("List should not include decrypted values")
		}
	}
}

func TestListByCategory(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Test", "")
	v.Set("identity.dob", "1990-01-01", "")
	v.Set("financial.income", "100k", "")

	fields, err := v.ListByCategory("identity")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 identity fields, got %d", len(fields))
	}
}

func TestGetByCategory(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Jane", "")
	v.Set("identity.dob", "1990-01-01", "")

	fields, err := v.GetByCategory("identity")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	// Should include values
	for _, f := range fields {
		if f.Value == "" {
			t.Fatal("GetByCategory should include decrypted values")
		}
	}
}

func TestGetContext(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Jane", "")
	v.Set("identity.dob", "1990-01-01", "")
	v.Set("financial.income", "100k", "")

	ctx, err := v.GetContext()
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(ctx.Categories))
	}
	if len(ctx.Categories["identity"]) != 2 {
		t.Fatalf("expected 2 identity fields, got %d", len(ctx.Categories["identity"]))
	}
	if len(ctx.Categories["financial"]) != 1 {
		t.Fatalf("expected 1 financial field, got %d", len(ctx.Categories["financial"]))
	}
}

func TestDelete(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Jane", "")
	v.Delete("identity.name")

	f, _ := v.Get("identity.name")
	if f != nil {
		t.Fatal("field should be deleted")
	}
}

func TestStatus(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Jane", "")
	v.Set("financial.income", "100k", "")

	status, err := v.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Locked {
		t.Fatal("should not be locked")
	}
	if !status.Initialized {
		t.Fatal("should be initialized")
	}
	if status.FieldCount != 2 {
		t.Fatalf("expected 2 fields, got %d", status.FieldCount)
	}
}

func TestValidateToken(t *testing.T) {
	v, _ := tmpVault(t)
	token := v.session.Token()

	if !v.ValidateToken(token) {
		t.Fatal("valid token should pass")
	}
	if v.ValidateToken("wrong-token") {
		t.Fatal("wrong token should fail")
	}
}

func TestAutoLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".pvault")
	sk, _ := Init(dir, testPassword)
	v, _ := Open(dir)
	defer v.Close()

	v.Unlock(testPassword, sk)

	// Override TTL to very short
	v.session.mu.Lock()
	v.session.ttl = 50 * time.Millisecond
	v.session.timer.Reset(50 * time.Millisecond)
	v.session.mu.Unlock()

	time.Sleep(150 * time.Millisecond)

	_, err := v.Get("anything")
	if err != ErrLocked {
		t.Fatalf("expected ErrLocked after auto-lock, got %v", err)
	}
}

func TestAuditLog(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.name", "Jane", "")
	v.Get("identity.name")

	entries, err := v.AuditLog(50)
	if err != nil {
		t.Fatal(err)
	}
	// At minimum: unlock + set write + set log + get read + get log
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 audit entries, got %d", len(entries))
	}
}

func TestSetSensitivity(t *testing.T) {
	v, _ := tmpVault(t)
	v.Set("identity.ssn", "123-45-6789", "standard")
	v.SetSensitivity("identity.ssn", "critical")

	f, _ := v.Get("identity.ssn")
	if f.Sensitivity != "critical" {
		t.Fatalf("expected critical, got %s", f.Sensitivity)
	}
}

func TestCreateServiceToken(t *testing.T) {
	v, _ := tmpVault(t)

	token, err := v.CreateServiceToken("life", "*", 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected non-empty service token")
	}
	if len(token) != 64 { // 32 bytes hex-encoded
		t.Fatalf("expected 64-char hex token, got %d chars", len(token))
	}
}

func TestValidateServiceToken(t *testing.T) {
	v, _ := tmpVault(t)

	token, _ := v.CreateServiceToken("life", "*", 24*time.Hour)

	svcToken, ok := v.ValidateServiceToken(token)
	if !ok {
		t.Fatal("expected valid service token")
	}
	if svcToken.Consumer != "life" {
		t.Fatalf("expected consumer 'life', got %q", svcToken.Consumer)
	}
	if svcToken.Usage != "service" {
		t.Fatalf("expected usage 'service', got %q", svcToken.Usage)
	}
}

func TestValidateServiceToken_Invalid(t *testing.T) {
	v, _ := tmpVault(t)

	_, ok := v.ValidateServiceToken("nonexistent-token")
	if ok {
		t.Fatal("expected invalid for nonexistent token")
	}
}

func TestValidateServiceToken_SessionTokenNotAccepted(t *testing.T) {
	v, _ := tmpVault(t)
	sessionToken := v.session.Token()

	_, ok := v.ValidateServiceToken(sessionToken)
	if ok {
		t.Fatal("session token should not validate as service token")
	}
}

func TestListServiceTokens(t *testing.T) {
	v, _ := tmpVault(t)
	v.CreateServiceToken("life", "*", 24*time.Hour)
	v.CreateServiceToken("other-app", "identity", 24*time.Hour)

	tokens, err := v.ListServiceTokens()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 service tokens, got %d", len(tokens))
	}
}

func TestRevokeServiceToken(t *testing.T) {
	v, _ := tmpVault(t)
	token, _ := v.CreateServiceToken("life", "*", 24*time.Hour)

	n, err := v.RevokeServiceToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 revoked, got %d", n)
	}

	_, ok := v.ValidateServiceToken(token)
	if ok {
		t.Fatal("revoked token should not validate")
	}
}

func TestServiceToken_RequiresUnlocked(t *testing.T) {
	v, _ := tmpVault(t)
	v.Lock()

	_, err := v.CreateServiceToken("life", "*", 24*time.Hour)
	if err != ErrLocked {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

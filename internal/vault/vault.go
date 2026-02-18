package vault

import (
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lovincyrus/personal-vault/internal/crypto"
	"github.com/lovincyrus/personal-vault/internal/store"
)

var (
	ErrLocked         = errors.New("vault is locked")
	ErrAlreadyUnlocked = errors.New("vault is already unlocked")
	ErrNotInitialized = errors.New("vault is not initialized")
	ErrAlreadyInit    = errors.New("vault is already initialized")
	ErrWrongPassword  = errors.New("wrong password or secret key")
	ErrInvalidTier    = errors.New("invalid sensitivity tier: must be public, standard, sensitive, or critical")
)

var validTiers = map[string]bool{
	"public": true, "standard": true, "sensitive": true, "critical": true,
}

// Vault is the main entry point for vault operations.
type Vault struct {
	mu      sync.RWMutex
	db      *store.DB
	session *Session
	dir     string // ~/.pvault
	salt    []byte // loaded on unlock, used for HKDF subkey derivation
}

// Open opens an existing vault database.
func Open(dir string) (*Vault, error) {
	dbPath := filepath.Join(dir, "vault.db")
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &Vault{db: db, dir: dir}, nil
}

// Init creates a new vault: generates salt, secret key, and stores verification ciphertext.
func Init(dir, password string) (secretKey string, err error) {
	// Create directory
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create vault dir: %w", err)
	}
	dbPath := filepath.Join(dir, "vault.db")
	if _, err := os.Stat(dbPath); err == nil {
		return "", ErrAlreadyInit
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("create database: %w", err)
	}
	defer db.Close()

	// Generate salt
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	// Generate secret key
	sk, err := crypto.GenerateSecretKey()
	if err != nil {
		return "", fmt.Errorf("generate secret key: %w", err)
	}

	// Store salt and secret key hash in DB
	if err := db.SetMeta("salt", base64.StdEncoding.EncodeToString(salt)); err != nil {
		return "", err
	}
	if err := db.SetMeta("secret_key_hash", hex.EncodeToString(crypto.HashSecretKey(sk))); err != nil {
		return "", err
	}

	// Derive vault key and create verification ciphertext
	vaultKey := crypto.DeriveVaultKey([]byte(password), sk, salt)
	verifyPlaintext := []byte("personal-vault-verification")
	verifyCipher, err := crypto.EncryptToBase64(vaultKey, verifyPlaintext)
	if err != nil {
		return "", fmt.Errorf("create verification: %w", err)
	}
	if err := db.SetMeta("verification", verifyCipher); err != nil {
		return "", err
	}

	// Write secret key file
	skPath := filepath.Join(dir, "secret.key")
	skHex := hex.EncodeToString(sk)
	if err := os.WriteFile(skPath, []byte(skHex+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write secret key: %w", err)
	}

	// Zero vault key
	for i := range vaultKey {
		vaultKey[i] = 0
	}

	return skHex, nil
}

// Unlock derives the vault key and creates a session.
func (v *Vault) Unlock(password string, secretKeyHex string) (token string, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.session != nil {
		return "", ErrAlreadyUnlocked
	}

	init, err := v.db.IsInitialized()
	if err != nil {
		return "", err
	}
	if !init {
		return "", ErrNotInitialized
	}

	// Load salt
	saltB64, err := v.db.GetMeta("salt")
	if err != nil {
		return "", err
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return "", fmt.Errorf("decode salt: %w", err)
	}

	// Decode secret key
	sk, err := hex.DecodeString(strings.TrimSpace(secretKeyHex))
	if err != nil {
		return "", fmt.Errorf("decode secret key: %w", err)
	}

	// Verify secret key hash
	storedHash, err := v.db.GetMeta("secret_key_hash")
	if err != nil {
		return "", err
	}
	actualHash := hex.EncodeToString(crypto.HashSecretKey(sk))
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(actualHash)) != 1 {
		return "", ErrWrongPassword
	}

	// Derive vault key
	vaultKey := crypto.DeriveVaultKey([]byte(password), sk, salt)

	// Verify with stored ciphertext
	verifyCipher, err := v.db.GetMeta("verification")
	if err != nil {
		return "", err
	}
	plaintext, err := crypto.DecryptFromBase64(vaultKey, verifyCipher)
	if err != nil {
		return "", ErrWrongPassword
	}
	if string(plaintext) != "personal-vault-verification" {
		return "", ErrWrongPassword
	}

	// Store salt for HKDF subkey derivation
	v.salt = salt

	// Create session
	session, err := NewSession(vaultKey, func() {
		v.mu.Lock()
		v.session = nil
		v.mu.Unlock()
	})
	if err != nil {
		return "", err
	}
	v.session = session

	// Zero local copy of vault key
	for i := range vaultKey {
		vaultKey[i] = 0
	}

	// Log access
	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "unlock"})

	return session.Token(), nil
}

// Lock destroys the session and zeroes the vault key.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.session != nil {
		v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "lock"})
		v.session.Destroy()
		v.session = nil
	}
}

// Status returns the current vault status.
func (v *Vault) Status() (*VaultStatus, error) {
	init, err := v.db.IsInitialized()
	if err != nil {
		return nil, err
	}

	status := &VaultStatus{
		Initialized: init,
		Locked:      true,
	}

	v.mu.RLock()
	if v.session != nil {
		status.Locked = false
	}
	v.mu.RUnlock()

	if init {
		count, _ := v.db.FieldCount()
		status.FieldCount = count
		cats, _ := v.db.CategoryCounts()
		status.Categories = cats
	}

	return status, nil
}

// Set encrypts and stores a field value.
func (v *Vault) Set(id, value, sensitivity string) error {
	if err := ValidateFieldID(id); err != nil {
		return err
	}

	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return err
	}

	parts := strings.SplitN(id, ".", 2)
	category, fieldName := parts[0], parts[1]

	// Derive category subkey
	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, category)
	if err != nil {
		return fmt.Errorf("derive subkey: %w", err)
	}

	// Encrypt
	encrypted, err := crypto.EncryptToBase64(subkey, []byte(value))
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	if sensitivity == "" {
		sensitivity = "standard"
	}
	if !validTiers[sensitivity] {
		return ErrInvalidTier
	}

	err = v.db.SetField(store.Field{
		ID:          id,
		Category:    category,
		FieldName:   fieldName,
		Value:       encrypted,
		Sensitivity: sensitivity,
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		return err
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "write"})
	return nil
}

// Get decrypts and returns a field value.
func (v *Vault) Get(id string) (*FieldInfo, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	f, err := v.db.GetField(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, nil
	}

	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, f.Category)
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.DecryptFromBase64(subkey, f.Value)
	if err != nil {
		return nil, fmt.Errorf("decrypt field %s: %w", id, err)
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "read"})

	return &FieldInfo{
		ID:          f.ID,
		Category:    f.Category,
		FieldName:   f.FieldName,
		Value:       string(plaintext),
		Sensitivity: f.Sensitivity,
		UpdatedAt:   f.UpdatedAt,
		Version:     f.Version,
	}, nil
}

// List returns all field metadata (no values).
func (v *Vault) List() ([]FieldInfo, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}

	fields, err := v.db.ListFields()
	if err != nil {
		return nil, err
	}

	result := make([]FieldInfo, len(fields))
	for i, f := range fields {
		result[i] = FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		}
	}
	return result, nil
}

// ListByCategory returns field metadata for a category (no values).
func (v *Vault) ListByCategory(category string) ([]FieldInfo, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}

	fields, err := v.db.ListFieldsByCategory(category)
	if err != nil {
		return nil, err
	}

	result := make([]FieldInfo, len(fields))
	for i, f := range fields {
		result[i] = FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		}
	}
	return result, nil
}

// GetByCategory returns all decrypted fields for a category.
func (v *Vault) GetByCategory(category string) ([]FieldInfo, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	fields, err := v.db.GetFieldsByCategory(category)
	if err != nil {
		return nil, err
	}

	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, category)
	if err != nil {
		return nil, err
	}

	result := make([]FieldInfo, len(fields))
	for i, f := range fields {
		plaintext, err := crypto.DecryptFromBase64(subkey, f.Value)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", f.ID, err)
		}
		result[i] = FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Value:       string(plaintext),
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		}
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: category + ".*", Action: "read"})
	return result, nil
}

// GetContext returns all decrypted fields grouped by category.
func (v *Vault) GetContext() (*ContextBundle, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	fields, err := v.db.GetAllFields()
	if err != nil {
		return nil, err
	}

	bundle := &ContextBundle{Categories: make(map[string][]FieldInfo)}
	subkeys := make(map[string][]byte)

	for _, f := range fields {
		sk, ok := subkeys[f.Category]
		if !ok {
			sk, err = crypto.DeriveSubkey(vaultKey, v.salt, f.Category)
			if err != nil {
				return nil, err
			}
			subkeys[f.Category] = sk
		}

		plaintext, err := crypto.DecryptFromBase64(sk, f.Value)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", f.ID, err)
		}

		bundle.Categories[f.Category] = append(bundle.Categories[f.Category], FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Value:       string(plaintext),
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		})
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "context"})
	return bundle, nil
}

// Delete removes a field.
func (v *Vault) Delete(id string) error {
	if _, err := v.requireUnlocked(); err != nil {
		return err
	}

	if err := v.db.DeleteField(id); err != nil {
		return err
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "delete"})
	return nil
}

// SetSensitivity updates a field's sensitivity tier.
func (v *Vault) SetSensitivity(id, tier string) error {
	if _, err := v.requireUnlocked(); err != nil {
		return err
	}
	if !validTiers[tier] {
		return ErrInvalidTier
	}
	return v.db.SetSensitivity(id, tier)
}

// AuditLog returns recent audit entries.
func (v *Vault) AuditLog(limit int) ([]store.AuditEntry, error) {
	return v.db.GetAuditLog(limit)
}

// ValidateToken checks a session token.
func (v *Vault) ValidateToken(token string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session == nil {
		return false
	}
	return v.session.ValidateToken(token)
}

// hashServiceToken returns the hex-encoded SHA-256 hash of a token.
func hashServiceToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CreateServiceToken generates a long-lived service token for a consumer.
// The raw token is returned to the caller; only the SHA-256 hash is stored.
func (v *Vault) CreateServiceToken(consumer, scope string, ttl time.Duration) (string, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return "", err
	}

	tokenBytes := make([]byte, 32)
	if _, err := crand.Read(tokenBytes); err != nil {
		return "", err
	}
	tokenStr := hex.EncodeToString(tokenBytes)

	t := store.Token{
		TokenStr:  hashServiceToken(tokenStr),
		Consumer:  consumer,
		Scope:     scope,
		ExpiresAt: time.Now().Add(ttl),
		Usage:     "service",
		CreatedAt: time.Now(),
	}
	if err := v.db.CreateToken(t); err != nil {
		return "", err
	}

	v.db.LogAccess(store.AuditEntry{
		Consumer: "vault",
		Scope:    scope,
		Action:   "create_service_token",
		Purpose:  "consumer: " + consumer,
	})

	return tokenStr, nil
}

// ValidateServiceToken checks a service token by hashing it and looking up the hash.
func (v *Vault) ValidateServiceToken(token string) (*store.Token, bool) {
	t, err := v.db.GetToken(hashServiceToken(token))
	if err != nil || t == nil {
		return nil, false
	}
	if t.Usage != "service" {
		return nil, false
	}
	return t, true
}

// ListServiceTokens returns all service tokens.
func (v *Vault) ListServiceTokens() ([]store.Token, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}
	return v.db.ListTokensByUsage("service")
}

// RevokeServiceToken removes a service token by its hash.
func (v *Vault) RevokeServiceToken(token string) (int64, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return 0, err
	}
	n, err := v.db.DeleteToken(hashServiceToken(token))
	if err != nil {
		return 0, err
	}
	if n > 0 {
		v.db.LogAccess(store.AuditEntry{
			Consumer: "vault",
			Scope:    "*",
			Action:   "revoke_service_token",
		})
	}
	return n, nil
}

// TouchSession resets the auto-lock timer.
func (v *Vault) TouchSession() {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session != nil {
		v.session.Touch()
	}
}

// Close closes the database.
func (v *Vault) Close() error {
	v.Lock()
	return v.db.Close()
}

// LogAccess writes an entry to the audit log.
func (v *Vault) LogAccess(entry store.AuditEntry) {
	v.db.LogAccess(entry)
}

func (v *Vault) requireUnlocked() ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session == nil {
		return nil, ErrLocked
	}
	v.session.Touch()
	key := v.session.VaultKey()
	if key == nil {
		return nil, ErrLocked
	}
	return key, nil
}

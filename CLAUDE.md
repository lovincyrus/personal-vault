# Personal Vault

## Quick Reference

- **Language:** Go 1.26+
- **Database:** `modernc.org/sqlite` (pure Go, no CGO)
- **Crypto:** `golang.org/x/crypto` (argon2, hkdf) + stdlib `crypto/aes`, `crypto/cipher`
- **HTTP:** stdlib `net/http`
- **Tests:** `go test -race ./...`
- **Build:** `make build` (outputs `bin/pvault`)

## Commands

```sh
make build                        # Build binary to bin/pvault
make test                         # Run all tests with race detector
make clean                        # Remove build artifacts
go test -race ./internal/crypto/  # Test crypto layer only
go test -race ./internal/store/   # Test store layer only
go test -race ./internal/vault/   # Test vault core only
go test -race ./internal/api/     # Test HTTP API only
```

## CLI Usage

```sh
pvault init                              # Create new vault (~/.pvault/)
pvault unlock                            # Start background server
pvault lock                              # Stop server, zero keys
pvault serve                             # Foreground server (debugging)
pvault set <category.field> <value>      # Set encrypted field
pvault get <category.field>              # Get decrypted field
pvault list [category]                   # List fields
pvault delete <category.field>           # Delete field
pvault set-sensitivity <id> <tier>       # Set sensitivity tier
pvault export                            # Export all fields as JSON
pvault audit                             # Show access audit log
pvault status                            # Show vault status
```

## Architecture

```
cmd/pvault/      CLI (thin HTTP clients to the vault server)
internal/
  crypto/        KDF (Argon2id), cipher (AES-256-GCM), HKDF subkeys
  store/         SQLite CRUD (fields, documents, tokens, audit, meta)
  vault/         Business logic (init, unlock/lock, encrypt/decrypt, session)
  api/           HTTP server, handlers, Bearer token middleware
```

## Security Model

```
Profile Password + Secret Key (128-bit)
  → Argon2id (64MB, 3 iter) → Vault Key (256-bit, in-memory only)
  → HKDF per category → Category Subkeys
  → AES-256-GCM per field (12-byte random nonce)
```

- App-layer encryption: encrypt before INSERT, decrypt after SELECT
- Vault key exists only in memory while unlocked
- Auto-lock after 30 min idle
- Session token: 32 bytes crypto/rand, constant-time comparison
- Secret key at `~/.pvault/secret.key` (0600), never in database

## Conventions

- Field IDs are `category.field_name` (e.g., `identity.full_name`)
- Sensitivity tiers: `public`, `standard`, `sensitive`, `critical`
- All timestamps stored as RFC3339 strings in SQLite
- WAL mode enabled, busy_timeout=5000ms
- No CGO — pure Go for portability

## Environment Variables

- `VAULT_DIR` — vault directory (default: `~/.pvault`)
- `VAULT_ADDR` — server address for CLI (default: `http://127.0.0.1:7200`)
- `VAULT_PORT` — server port for `pvault serve` (default: `7200`)

## Testing

Tests use temp directories — no cleanup needed. All tests run with `-race`.

```go
// Pattern: create temp vault, run test
func tmpVault(t *testing.T) (*Vault, string) {
    dir := filepath.Join(t.TempDir(), ".vault")
    sk, _ := Init(dir, "test-password")
    v, _ := Open(dir)
    v.Unlock("test-password", sk)
    return v, sk
}
```

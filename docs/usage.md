# Usage

## Quick Start

```sh
make build && make install

pvault onboard               # Create vault, unlock, populate common fields (interactive)
pvault set identity.full_name "Cool Cucumber"
pvault get identity.full_name
pvault list
pvault lock                  # Stop server, zero keys from memory
```

Or step by step:

```sh
pvault init           # Create vault at ~/.pvault/, sets profile password + secret key
pvault unlock         # Start server at localhost:7200, prompts for password
```

Save your secret key somewhere safe. You need both the profile password and the secret key to unlock the vault.

## Fields

Fields use dot notation: `category.field_name`.

```sh
pvault set identity.full_name "Cool Cucumber"
pvault set identity.date_of_birth "1995-06-15"
pvault set addresses.home_city "San Francisco"
pvault set employment.employer "Acme Corp"
pvault set financial.filing_status "single"
pvault set preferences.timezone "America/Los_Angeles"
pvault get identity.full_name
pvault list                      # All fields
pvault list identity             # One category
pvault delete identity.date_of_birth
pvault export                    # All fields as JSON
```

You can use any category and field name. Run `pvault schema` to see recommended field names and their default sensitivity tiers.

## Sensitivity Tiers

Each field has a sensitivity tier that controls how it's shared with consumers.

| Tier | Examples | Behavior |
|------|----------|----------|
| `public` | Name, timezone, language | Auto-shared with authorized consumers |
| `standard` | Address, employer, education | Shared on request, logged |
| `sensitive` | DOB, phone, tax status | Requires explicit approval |
| `critical` | SSN, card number, card expiry | Requires approval + verification |

```sh
pvault set-sensitivity financial.ssn critical
pvault set-sensitivity preferences.timezone public
```

Default is `standard` for new fields. The recommended schema provides sensible defaults — use `pvault schema` to see them.

## Service Tokens

Service tokens let applications authenticate with the vault using long-lived credentials. They follow the 1Password service account pattern.

```sh
pvault create-service-token myapp --scope "*" --ttl 8760h
pvault create-service-token tax-agent --scope "identity.*,financial.*" --ttl 1h
pvault list-service-tokens
pvault revoke-service-token abc123    # Revoke by token prefix
```

Service tokens keep the vault alive. Each authenticated request resets the 30-minute auto-lock timer, so the vault stays unlocked as long as a consumer is active.

## HTTP API

The vault runs at `http://127.0.0.1:7200`. All protected endpoints require `Authorization: Bearer <token>`.

### Public

```
GET  /vault/status                       # { initialized, locked, field_count, categories }
GET  /vault/schema                       # Recommended field names and sensitivity tiers
POST /vault/unlock                       # { password, secret_key } → { token }
```

### Fields

```
GET    /vault/fields                     # List all field metadata (no values)
GET    /vault/fields/{id}                # Get field with decrypted value
PUT    /vault/fields/{id}                # { value, sensitivity? } — upsert
DELETE /vault/fields/{id}                # Delete field
GET    /vault/fields/category/{name}     # All fields in category with values
```

### Context

```
GET /vault/context                       # Full decrypted dump grouped by category
```

This is what consumers call. Returns:

```json
{
  "categories": {
    "identity": [
      { "id": "identity.full_name", "category": "identity", "field_name": "full_name", "value": "Cool Cucumber", "sensitivity": "standard" }
    ],
    "preferences": [...]
  }
}
```

### Sensitivity

```
PUT /vault/sensitivity/{id}              # { tier } — update sensitivity
```

### Service Tokens

```
POST   /vault/tokens/service             # { consumer, scope, ttl } → { token, expires_at }
GET    /vault/tokens/service             # List active tokens (values truncated)
DELETE /vault/tokens/service/{prefix}    # Revoke by prefix
```

### Session

```
POST /vault/lock                         # Lock vault, zero keys
```

### Audit

```
GET /vault/audit?limit=50                # Recent access log
```

## Security Model

```
Profile Password + Secret Key (128-bit)
  → Argon2id KDF (64MB, 3 iterations)
  → Vault Key (256-bit, in-memory only)
  → HKDF per category → Category Subkeys
  → AES-256-GCM per field (12-byte random nonce)
```

- Profile password is never stored
- Secret key lives at `~/.pvault/secret.key` (mode 0600), never transmitted
- Vault key exists only in memory while unlocked, zeroed on lock
- Auto-lock after 30 minutes of inactivity
- Every access logged to `vault_access_log`

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `VAULT_DIR` | `~/.pvault` | Vault directory |
| `VAULT_ADDR` | `http://127.0.0.1:7200` | Server address for CLI |
| `VAULT_PORT` | `7200` | Server listen port |

## File Layout

```
~/.pvault/
├── vault.db       # SQLite database (encrypted fields)
├── secret.key     # 128-bit secret key (mode 0600)
├── .session       # Session token (created on unlock)
└── pvault.pid     # PID of running server
```

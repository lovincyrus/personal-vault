<h1 align="center">Personal Vault</h1>

**An encrypted vault for personal context. You hold the keys. The reference implementation of the [Personal Context Protocol](https://github.com/lovincyrus/personal-context-protocol).**

---

You fill out your life once. Every AI agent starts with context, not a blank slate.

Personal Vault stores your identity, documents, relationships, financial data, addresses, and preferences — encrypted per-field with keys derived from your profile password. Agents request scoped access. You approve. The token expires. Your keys never leave your device.

## Usage

```sh
curl -fsSL https://www.personalvault.dev/install.sh | sh
pvault onboard
```

Save your secret key somewhere safe. You need both the profile password and the secret key to unlock.

<details>
<summary>Alternative: build from source</summary>

```sh
git clone https://github.com/lovincyrus/personal-vault.git
cd personal-vault && make build && make install
pvault onboard
```
</details>

<details>
<summary>Alternative: install with Go</summary>

```sh
go install github.com/lovincyrus/personal-vault/cmd/pvault@latest
pvault onboard
```
</details>

### Manual setup

```sh
pvault init                          # Create vault, set profile password + secret key
pvault unlock                        # Start server at localhost:7200
pvault set identity.full_name "Cool Cucumber"
pvault set addresses.current.city "Seattle"
pvault set financial.filing_status "single"
pvault get identity.full_name        # Cool Cucumber
pvault list                          # All fields
pvault lock                          # Stop server, zero keys from memory
```

## How it works

```
Profile Password + Secret Key (128-bit)
  → Argon2id KDF (64MB, 3 iterations)
  → Vault Key (256-bit, in-memory only)
  → HKDF per category → Category Subkeys
  → AES-256-GCM per field (12-byte random nonce)
```

The profile password is never stored. The secret key lives on your device. The vault key exists only in memory while unlocked. Each field is encrypted individually. If someone steals the database, they get ciphertext.

## CLI

```sh
pvault init                              # Create a new vault
pvault unlock                            # Unlock (starts background server)
pvault lock                              # Lock (stops server, zeroes keys)
pvault status                            # Show vault status

pvault set <id> <value>                  # Set a field
pvault get <id>                          # Get a field
pvault list [category]                   # List fields
pvault delete <id>                       # Delete a field
pvault export                            # Export all fields as JSON

pvault set-sensitivity <id> <tier>       # Set sensitivity tier
pvault audit                             # Show access log

pvault create-service-token <consumer>   # Create a long-lived token
pvault list-service-tokens               # List active tokens
pvault revoke-service-token <prefix>     # Revoke a token by prefix
```

Fields use dot notation: `identity.full_name`, `addresses.current.city`, `financial.filing_status`. You can use any category and field name.

## HTTP API

The vault runs at `http://127.0.0.1:7200`. Protected endpoints require `Authorization: Bearer <token>`.

```
GET    /vault/status                    # Vault status (public)
POST   /vault/unlock                    # Unlock → session token

GET    /vault/fields                    # List field metadata
GET    /vault/fields/{id}               # Get field with decrypted value
PUT    /vault/fields/{id}               # Set field
DELETE /vault/fields/{id}               # Delete field
GET    /vault/fields/category/{name}    # All fields in a category

GET    /vault/context                   # Full decrypted dump by category

PUT    /vault/sensitivity/{id}          # Update sensitivity tier

POST   /vault/tokens/service            # Create service token
GET    /vault/tokens/service            # List service tokens
DELETE /vault/tokens/service/{prefix}   # Revoke service token

POST   /vault/lock                      # Lock vault
GET    /vault/audit                     # Access audit log
```

## Service tokens

Service tokens let applications authenticate with the vault using long-lived, scoped credentials.

```sh
pvault create-service-token tax-agent --scope "identity.*,financial.*" --ttl 1h
pvault create-service-token life --scope "*" --ttl 8760h
pvault list-service-tokens
pvault revoke-service-token abc123
```

Each authenticated request resets the 30-minute auto-lock timer.

## Sensitivity tiers

| Tier | Examples | Behavior |
|------|----------|----------|
| `public` | Name, timezone | Auto-shared with authorized consumers |
| `standard` | Address, employer | Shared on request, logged |
| `sensitive` | DOB, passport number | Requires explicit approval |
| `critical` | SSN, bank routing | Requires approval + verification |

```sh
pvault set-sensitivity identity.ssn critical
pvault set-sensitivity preferences.timezone public
```

Default is `standard` for new fields.

## Architecture

```
~/.pvault/
├── vault.db       # SQLite (encrypted fields, audit log)
├── secret.key     # 128-bit secret key (mode 0600)
├── .session       # Session token (created on unlock)
└── pvault.pid     # PID of running server
```

Pure Go. No CGO. Single binary. Runs on localhost. SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite).

## Tests

```sh
make test    # 76 tests, race detector enabled
```

## MCP server

The vault ships with a TypeScript [MCP](https://modelcontextprotocol.io/) server so AI agents can access your personal context.

```sh
claude mcp add vault -- npx -y personal-vault-mcp@latest
```

| Tool | Description |
|------|-------------|
| `vault_status` | Check if the vault is running and unlocked |
| `vault_get` | Get a single decrypted field by ID |
| `vault_list` | List all field metadata (no values) |
| `vault_context` | Get all decrypted fields grouped by category |
| `vault_set` | Set an encrypted field value in the vault |

See [mcp/README.md](./mcp/README.md) for authentication, environment variables, and error messages.

## Shopping demo

A self-contained demo showing two MCP servers working together: the vault provides personal context, a mock shop handles orders. One sentence in, order confirmation out — no questions asked.

```sh
cd examples/shopping-demo && npm install
claude mcp add shop -- npx tsx examples/shopping-demo/src/index.ts
```

> Test the shopping demo — order a t-shirt using my vault data.

The agent reads your name, email, and address from the vault, browses the shop, and places the order. If it discovers missing information (like t-shirt size), it asks you and offers to save it to the vault for next time.

See [examples/shopping-demo/README.md](./examples/shopping-demo/README.md) for full setup instructions.

## The protocol

This is the reference implementation of the **[Personal Context Protocol](https://github.com/lovincyrus/personal-context-protocol)** — an open protocol for AI agents to access personal context. Read the [specification](https://github.com/lovincyrus/personal-context-protocol/blob/main/specification.md).

## Full documentation

See [docs/usage.md](./docs/usage.md) for the complete API reference, environment variables, and integration guide.

## License

[MIT](./LICENSE)

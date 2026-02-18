# Vault MCP Server

MCP server that exposes your encrypted personal vault to AI agents.

## Install

```sh
npm install personal-vault-mcp
```

Or run directly with npx:

```sh
npx -y personal-vault-mcp@latest
```

## Register with Claude Code

```sh
claude mcp add vault -- npx -y personal-vault-mcp@latest
```

## Tools

| Tool | Description |
|------|-------------|
| `vault_status` | Check if the vault is running and unlocked |
| `vault_get` | Get a single decrypted field by ID |
| `vault_list` | List all field metadata (no values) |
| `vault_context` | Get all decrypted fields grouped by category |
| `vault_set` | Set an encrypted field value in the vault |

## Authentication

The MCP server authenticates with the vault automatically:

1. `VAULT_TOKEN` env var — use with service tokens for always-on agents
2. `~/.pvault/.session` — session token from `pvault unlock`

Token is resolved on each request, so the server survives vault lock/unlock cycles.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VAULT_ADDR` | `http://127.0.0.1:7200` | Vault server address |
| `VAULT_DIR` | `~/.pvault` | Vault directory (for session file) |
| `VAULT_TOKEN` | — | Service token (overrides session file) |

## Error Messages

| Error | Meaning |
|-------|---------|
| `vault: server not running` | Run `pvault unlock` to start the server |
| `vault: session expired` | Run `pvault unlock` to refresh the session |
| `vault: vault is locked` | Run `pvault unlock` to decrypt |
| `vault: not found` | Field ID doesn't exist |
| `vault: not configured` | No token available — run `pvault unlock` or set `VAULT_TOKEN` |

## Demo

See [examples/shopping-demo](../examples/shopping-demo/) for a demo using Vault MCP + a mock shop.

# Last Mile: Pre-Launch Checklist

The code and security model are solid. The gap is the distance between "I'm curious" and "I'm holding it." Every step a user has to take before they feel the vault working is a step where they might walk away.

## Current Setup Friction

What a user has to do today:

1. Have Go installed (most people don't)
2. Clone the repo
3. `make build && make install` (needs `/usr/local/bin` write access)
4. `pvault init` — save the secret key (shown once, easy to miss)
5. `pvault unlock`
6. Manually `pvault set` 9+ fields to populate data
7. `cd mcp && npm install` (requires Node.js)
8. `claude mcp add vault -- npx tsx mcp/src/index.ts`
9. Then they can try the demo

That's 9 steps across 2 toolchains before anyone experiences the product.

## Priority Gaps

### P0: Zero-dependency install

**Problem:** Users must have Go to compile from source.

**Fix:** GitHub Releases with prebuilt binaries (goreleaser). Provide a one-liner:

```sh
curl -fsSL https://... | sh
```

Targets: macOS arm64, macOS amd64, Linux arm64, Linux amd64.

### P0: Homebrew tap

**Problem:** macOS developers expect `brew install`.

**Fix:** Create a Homebrew tap (`yourusername/tap/personal-vault`). goreleaser can generate the formula automatically.

### P1: `pvault onboard` command

**Problem:** Users must run 9+ `pvault set` commands before anything interesting happens.

**Fix:** Interactive wizard that does init + unlock + populates common fields in one pass:

```
$ pvault onboard
Create your vault
  Profile password: ********
  Confirm: ********

Your secret key (save this somewhere safe):
  XXXXX-XXXXX-XXXXX-XXXXX

Let's add some basics (press Enter to skip any):
  Full name: Cool Cucumber
  Email: cool@example.com
  Street: 123 Main St
  City: Seattle
  State: WA
  ZIP: 98101
  Country: US

Vault ready. 7 fields saved.
```

### P1: Bundle the MCP server

**Problem:** MCP requires a separate `npm install` — second toolchain, second failure point.

**Options:**
- Compile TS into the Go binary (embed + subprocess)
- `pvault mcp` subcommand that serves the MCP protocol directly in Go
- Distribute a self-contained Node script with no install step

Best option: native Go MCP server (`pvault mcp`). Eliminates Node.js dependency entirely.

### P2: Rename the binary

**Problem:** `vault` collides with HashiCorp Vault, which is installed on many developer machines.

**Decision:** Renamed to `pvault`.

## Dream Flow

```sh
brew install yourusername/tap/personal-vault
pvault onboard
# → vault initialized, unlocked, populated with basics

claude mcp add vault -- pvault mcp
# → MCP connected, agent can use the vault immediately
```

Three commands. Under 2 minutes. Puppy in their hands.

## Reference

- "Put the puppy in their hands" — https://kern.io/p/puppy-in-their-hands
- goreleaser: https://goreleaser.com
- Homebrew taps: https://docs.brew.sh/Taps

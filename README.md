# Forge

A ground-up, interop-first alternative to opencode: a Go daemon that is wire-compatible with opencode's HTTP+SSE+WebSocket API.

opencode is amazing. This is a personal exercise in reimplementing it from scratch in Go — to see how far the rewrite gets, and as a bonus ship the mobile client that opencode is missing.

Wire compatibility is kept deliberately until interop becomes a wall worth breaking.

## Architecture

```
                ┌─────────────────────────────────────────────┐
   Mobile  ─────┤   Forge Daemon (Go, single static binary)   │── SQLite (sessions/msgs/parts)
   (primary) ───┤   - HTTP/REST + SSE bus + WS PTY            │── repo + built-in tools
   TUI (Go) ────┤   - Auth + directory/instance routing       │── MCP clients (stdio/http/sse)
   opencode's   │   - Agent engine (LLM stream + tool loop)   │── LSP servers (jsonrpc)
   web/desktop  │   - Ecosystem loaders                       │
   (unmodified) │   - Plugin host sidecar (Node/Bun) ◄────────┼── opencode-format TS plugins
                └─────────────────────────────────────────────┘
        all clients speak the SAME opencode wire protocol
```

All clients — mobile, Go TUI, and unmodified opencode web/desktop — speak the same wire protocol (~113 REST+SSE+WebSocket endpoints).

## Quick Start

**Prerequisites:** Go 1.22+

```sh
git clone https://github.com/rotemmiz/forge
cd forge
make build          # outputs bin/forged
./bin/forged serve
```

The daemon listens on `localhost:4096` by default. Clients authenticate via HTTP Basic or `?auth_token=base64(user:pass)` and route to per-directory instances using the `x-opencode-directory` header.

## Testing

```sh
make test                 # unit tests
make conformance          # conformance suite against TARGET= (default: localhost:4096)
make selfdiff             # opencode-vs-opencode self-diff gate
```

Or directly:

```sh
go test ./...
scripts/run-conformance.sh self
```

## Project Status

Early development. The wire protocol conformance harness is the correctness gate. Passing `make selfdiff` clean is required before merging any change that touches an API endpoint.

## License

MIT — see [LICENSE](LICENSE).

## Further Reading

- [`plans/00-masterplan.md`](plans/00-masterplan.md) — vision, frozen wire contract, build sequence
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — git workflow, local CI gate, code style
- [`DEPENDENCIES.md`](DEPENDENCIES.md) — vetted library choices and rationale

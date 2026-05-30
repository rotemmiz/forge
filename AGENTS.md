# Forge Agent Guidelines

This document provides high-signal instructions for working with the Forge repository.

## Project Overview
- Forge is a Go daemon, wire-compatible with opencode's HTTP+SSE+WebSocket API.
- The primary client is mobile (Android).
- The `plans/` directory contains detailed engineering plans and is the source of truth for design.

## Key Developer Commands

### Build
- `go build -o bin/forged ./cmd/forged`: Build the Forge daemon.

### Testing & Verification
- `go test ./...`: Run all unit tests.
- `go test ./conformance/... -target=<url>`: Run the conformance suite against a specified target URL.
- `go run ./conformance/cmd/record -url <url> -out conformance/cassettes/sse-catalog.json`: Record opencode truth cassettes (requires a running opencode instance).
- `bash scripts/run-conformance.sh self`: Execute the opencode-vs-opencode conformance self-diff gate.

### Code Quality & Generation
- `golangci-lint run`: Run static analysis/linting.
- `gofmt -l .`: Check Go code formatting.
- `go generate ./...`: Regenerate code (e.g., from OpenAPI spec using `oapi-codegen`).
- `go mod tidy`: Clean up `go.mod` and `go.sum` files.
- `bash scripts/sync-openapi.sh`: Sync OpenAPI specification.
- `bash scripts/check-spec-drift.sh`: Check for OpenAPI spec drift.

## Git Workflow & Local Review Gate

**IMPORTANT: Hosted CI minutes are exhausted. Rely on local review checks.**

Before opening a PR, run the following checks locally in order:
1.  `go build ./...` (or `go build/vet`)
2.  `gofmt -l .`
3.  `golangci-lint run`
4.  `go test ./...`
5.  `go generate ./...` followed by `git diff --exit-code internal/api/gen/` to ensure generated code is up-to-date.
6.  `bash scripts/run-conformance.sh self`

## Architectural Notes
- **Wire-compatibility:** Strict adherence to opencode's wire protocol is a non-negotiable.
- **Reference Codebase:** The original opencode daemon is located at `/Users/rotemmiz/git/opencode`.
- **Frozen Contract:** The OpenAPI specification is at `/Users/rotemmiz/git/opencode/packages/sdk/openapi.json`.
- **Plugin Host:** TypeScript/JavaScript plugins require a Node/Bun sidecar (see `plans/05-plugin-host.md`).
- **API Versioning:** Pin to v2 APIs; v1 endpoints are best-effort.

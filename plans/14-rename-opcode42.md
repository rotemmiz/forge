# Plan 14 — Rename: Forge → Opcode42 (DONE)

The project brand was renamed from **Forge** to **Opcode42** across every layer —
Go daemon, Android app, generated SDKs, build/release, packaging, and docs — while
**every opencode wire-compat identifier was preserved** (`x-opencode-*`,
`_opencode._tcp`, the frozen `openapi-reference.json`, `OPENCODE_*` suffixes).

- **Executed in PR #142** (the full source-grounded rename) and **PR #143**
  (repo/registry seam flip + teardown of the temporary rename scaffolding).
- GitHub repo renamed `rotemmiz/forge` → `rotemmiz/opcode42`; the Go module is
  `github.com/rotemmiz/opcode42`.

**Two-family naming convention** (documented here so it isn't "corrected" later):
- **Brand identifiers keep the `42`**: `Opcode42*` exported Go types, `dev.opcode42.*`
  Android/Kotlin packages, the `.opcode42` config/state dir, `opcode42.local` mDNS host.
- **CLI/ops drop the `42`**: the `opcoded` daemon binary, the `opcode-tui` binary, the
  `OPCODE_` env-var prefix.

The full original plan — before→after tables, the ordered replacement recipe, the
phase-by-phase execution log, and the `check-rename.sh` acceptance gate — lives in
**git history** (see PR #142). The acceptance gate and its CI job were intentionally
removed once the rename was complete and verified.

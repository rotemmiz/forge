// Package forgeclient is the Go SDK for the Forge / opencode wire protocol: the
// generated REST client (sub-package gen) wrapped with auth + directory-routing
// header injection, plus hand-written SSE (sse.go) and WebSocket-PTY (pty.go)
// clients that codegen cannot express.
//
// It is wire-generic — point it at a Forge daemon or a real opencode daemon; the
// contract is identical. Used by the Go TUI (plan 08), integration tests, and the
// conformance harness.
package forgeclient

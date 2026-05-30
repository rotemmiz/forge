// Package forgeclient is the Go SDK for the Forge / opencode wire protocol: the
// generated REST client (sub-package gen) wrapped with auth + directory-routing
// header injection, plus a hand-written SSE client (sse.go) that codegen cannot
// express. A WebSocket-PTY client is a forthcoming addition (plan 06; needed
// only for the TUI's optional PTY pane).
//
// It is wire-generic — point it at a Forge daemon or a real opencode daemon; the
// contract is identical. Used by the Go TUI (plan 08), integration tests, and the
// conformance harness.
package forgeclient

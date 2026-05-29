package id

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Prefix is a typed ID namespace. Values mirror opencode's prefixes map exactly
// so IDs are byte-compatible on the wire (id/id.ts:3-14).
type Prefix string

// Prefixes for each entity. Keep in sync with opencode id/id.ts:3-14.
const (
	Job        Prefix = "job"
	Event      Prefix = "evt"
	Session    Prefix = "ses"
	Message    Prefix = "msg"
	Permission Prefix = "per"
	Question   Prefix = "que"
	Part       Prefix = "prt"
	PTY        Prefix = "pty"
	Tool       Prefix = "tool"
	Workspace  Prefix = "wrk"
)

// length is the total character count of the random suffix portion, matching
// opencode's LENGTH=26 (id/id.ts:16): 12 hex time chars + 14 base62 chars.
const length = 26

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// state guards the monotonic counter so concurrent generation stays ordered
// within a millisecond (opencode relies on single-threaded JS; Go needs a lock).
var state struct {
	mu            sync.Mutex
	lastTimestamp int64
	counter       int64
}

// Ascending returns an ID that sorts in chronological creation order.
func Ascending(p Prefix) string { return create(string(p), false, 0) }

// Descending returns an ID that sorts newest-first. Sessions use this so list
// queries return the most recent first (matching opencode's session ordering).
func Descending(p Prefix) string { return create(string(p), true, 0) }

// create builds a prefixed ID: "<prefix>_" + 12 hex chars encoding
// (timestamp*0x1000 + counter) over 6 bytes + 14 random base62 chars.
// When descending, the 48-bit time value is bit-inverted (id/id.ts:51-70).
func create(prefix string, descending bool, ts int64) string {
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}

	state.mu.Lock()
	if ts != state.lastTimestamp {
		state.lastTimestamp = ts
		state.counter = 0
	}
	state.counter++
	counter := state.counter
	state.mu.Unlock()

	now := ts*0x1000 + counter
	if descending {
		// Invert the low 48 bits (the 6 bytes we emit), matching JS bitwise ~
		// over the same range.
		now = ^now & 0xFFFFFFFFFFFF
	}

	timeHex := make([]byte, 6)
	for i := 0; i < 6; i++ {
		timeHex[i] = byte((now >> (40 - 8*i)) & 0xFF)
	}

	return fmt.Sprintf("%s_%x%s", prefix, timeHex, randomBase62(length-12))
}

func randomBase62(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand should never fail; if it does, IDs would collide, so panic.
		panic(fmt.Sprintf("id: crypto/rand failed: %v", err))
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = base62[int(buf[i])%62]
	}
	return string(out)
}

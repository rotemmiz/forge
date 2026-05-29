package enginetest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rotemmiz/forge/internal/bus"
	"github.com/rotemmiz/forge/internal/engine"
	"github.com/rotemmiz/forge/internal/engine/catalog"
	"github.com/rotemmiz/forge/internal/engine/llm"
	"github.com/rotemmiz/forge/internal/engine/message"
	"github.com/rotemmiz/forge/internal/engine/permission"
	"github.com/rotemmiz/forge/internal/engine/registry"
	"github.com/rotemmiz/forge/internal/engine/tool"
	"github.com/rotemmiz/forge/internal/storage"
)

// rig is a full engine wired over an in-memory store + bus + mock provider, the
// deterministic integration harness for the M9 text-only and tool-call gates.
type rig struct {
	eng       *engine.Engine
	store     *message.Store
	bus       *bus.Bus
	sub       <-chan bus.Event
	sessionID string
	mock      *MockProvider
}

func newRig(t *testing.T, scripts ...[]llm.Event) *rig {
	t.Helper()
	return newRigInDir(t, t.TempDir(), scripts...)
}

func newRigInDir(t *testing.T, dir string, scripts ...[]llm.Event) *rig {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const sessionID = "ses_e2e"
	if _, err := db.Exec(`INSERT INTO project (id, worktree, time_created, time_updated) VALUES ('p','/tmp',0,0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session (id, project_id, slug, directory, version, time_created, time_updated)
		VALUES (?, 'p','s','/tmp','1',0,0)`, sessionID); err != nil {
		t.Fatal(err)
	}
	store := message.NewStore(db)
	b := bus.NewInstanceBus(sessionID, nil)
	sub, _ := b.Subscribe()
	mock := NewMockProvider(scripts...)

	reg := registry.New(tool.Bash{}, tool.Read{}, tool.Write{}, tool.Edit{}, tool.Glob{}, tool.Grep{})
	eng := engine.New(engine.Config{
		Store:       store,
		Catalog:     catalog.Fixture(),
		Registry:    reg,
		Permissions: permission.NewManager(b),
		Bus:         b,
		Directory:   dir,
		// Allow all tools so the loop runs unattended in tests.
		Rulesets:  []permission.Ruleset{{{Permission: "*", Pattern: "*", Action: permission.ActionAllow}}},
		Providers: func(context.Context, string, string) (llm.Provider, error) { return mock, nil },
	})
	return &rig{eng: eng, store: store, bus: b, sub: sub, sessionID: sessionID, mock: mock}
}

func (r *rig) prompt(t *testing.T, text string) message.WithParts {
	t.Helper()
	out, err := r.eng.Prompt(context.Background(), engine.PromptInput{
		SessionID: r.sessionID, Provider: "openai", Model: "gpt-4o",
		Parts: []engine.PartInput{{Type: "text", Text: text}},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	return out
}

func (r *rig) drain() []bus.Event {
	var out []bus.Event
	for {
		select {
		case e := <-r.sub:
			out = append(out, e)
		default:
			return out
		}
	}
}

func eventTypes(events []bus.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Type)
	}
	return out
}

func countType(events []bus.Event, typ string) int {
	n := 0
	for _, e := range events {
		if e.Type == typ {
			n++
		}
	}
	return n
}

// Scenario 1: a single text prompt produces a streamed text response.
func TestE2E_TextOnly(t *testing.T) {
	script := NewScript().StepStart().Text("t1", "Hello", ", world").
		StepFinish("stop", llm.TokenUsage{Input: 10, Output: 5}).Finish().Events()
	r := newRig(t, script)

	out := r.prompt(t, "hi there")

	if out.Info.Assistant == nil || out.Info.Assistant.Finish != "stop" {
		t.Fatalf("assistant did not finish stop: %+v", out.Info.Assistant)
	}
	// Final assistant carries the streamed text.
	var text string
	for _, p := range out.Parts {
		if tp, ok := p.(*message.TextPart); ok {
			text += tp.Text
		}
	}
	if text != "Hello, world" {
		t.Fatalf("assistant text = %q, want %q", text, "Hello, world")
	}

	events := r.drain()
	// User message, assistant placeholder, streamed deltas, and a final message.updated.
	if countType(events, "message.part.delta") != 2 {
		t.Fatalf("want 2 part.delta, got %d (%v)", countType(events, "message.part.delta"), eventTypes(events))
	}
	if countType(events, "message.updated") < 2 {
		t.Fatalf("want >=2 message.updated (user + assistant), got %d", countType(events, "message.updated"))
	}
	if countType(events, "message.part.updated") == 0 {
		t.Fatalf("missing message.part.updated events")
	}

	// DB: exactly one user + one assistant message persisted.
	msgs, _ := r.store.List(context.Background(), r.sessionID)
	if len(msgs) != 2 || !msgs[0].Info.IsUser() || msgs[1].Info.Assistant == nil {
		t.Fatalf("want [user, assistant] persisted, got %d messages", len(msgs))
	}
	if r.mock.Calls() != 1 {
		t.Fatalf("want 1 provider call, got %d", r.mock.Calls())
	}
}

// Scenario 2: a tool call is executed, fed back, and a final answer streamed.
func TestE2E_ToolCall(t *testing.T) {
	dir := t.TempDir()
	// Step 1: model asks to write a file. Step 2: model answers with text.
	step1 := NewScript().StepStart().
		ToolCall("call_1", "write", map[string]any{"filePath": "out.txt", "content": "done"}).
		StepFinish("tool-calls", llm.TokenUsage{Input: 20, Output: 8}).Finish().Events()
	step2 := NewScript().StepStart().Text("t2", "Wrote the file.").
		StepFinish("stop", llm.TokenUsage{Input: 30, Output: 4}).Finish().Events()

	r := newRigInDir(t, dir, step1, step2)
	out := r.prompt(t, "write out.txt")

	if out.Info.Assistant == nil || out.Info.Assistant.Finish != "stop" {
		t.Fatalf("final assistant should finish stop: %+v", out.Info.Assistant)
	}
	if r.mock.Calls() != 2 {
		t.Fatalf("want 2 provider calls (tool round-trip), got %d", r.mock.Calls())
	}

	// The first assistant message holds a completed tool part.
	msgs, _ := r.store.List(context.Background(), r.sessionID)
	var sawCompletedTool bool
	for _, m := range msgs {
		for _, p := range m.Parts {
			if tp, ok := p.(*message.ToolPart); ok && tp.Tool == "write" && tp.Status() == message.ToolCompleted {
				sawCompletedTool = true
			}
		}
	}
	if !sawCompletedTool {
		t.Fatalf("no completed write tool part found")
	}
	// The second provider request must include the tool result in its messages.
	reqs := r.mock.Requests()
	if len(reqs) != 2 {
		t.Fatalf("want 2 requests, got %d", len(reqs))
	}
	if !hasToolResult(reqs[1].Messages) {
		t.Fatalf("second request missing tool-result message: %+v", reqs[1].Messages)
	}
	// And the file was actually written by the tool.
	if data, err := readFile(dir, "out.txt"); err != nil || strings.TrimSpace(data) != "done" {
		t.Fatalf("tool did not write file: %q err=%v", data, err)
	}
}

func hasToolResult(msgs []llm.ModelMessage) bool {
	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Kind == llm.ContentToolResult {
				return true
			}
		}
	}
	return false
}

func readFile(dir, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	return string(data), err
}

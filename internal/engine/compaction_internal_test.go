package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rotemmiz/forge/internal/engine/message"
	"github.com/rotemmiz/forge/internal/storage"
)

func user(id string) message.WithParts {
	u := &message.UserMessage{ID: id, SessionID: "s", Role: message.RoleUser}
	return message.WithParts{Info: message.Info{User: u},
		Parts: []message.Part{&message.TextPart{PartBase: message.PartBase{ID: "p" + id, SessionID: "s", MessageID: id}, Type: "text", Text: "hi"}}}
}

func assistant(id, parent string) message.WithParts {
	a := &message.AssistantMessage{ID: id, SessionID: "s", Role: message.RoleAssistant, ParentID: parent, Finish: "stop"}
	return message.WithParts{Info: message.Info{Assistant: a}}
}

func TestSelectTail(t *testing.T) {
	// 4 user turns (u1..u4); keep last 2 -> head is turns u1,u2.
	history := []message.WithParts{
		user("u1"), assistant("a1", "u1"),
		user("u2"), assistant("a2", "u2"),
		user("u3"), assistant("a3", "u3"),
		user("u4"), assistant("a4", "u4"),
	}
	head, tail := selectTail(history, 2)
	if tail != "u3" {
		t.Fatalf("tail start = %q, want u3", tail)
	}
	if len(head) != 4 { // u1,a1,u2,a2
		t.Fatalf("head len = %d, want 4", len(head))
	}
	if head[0].Info.ID() != "u1" || head[3].Info.ID() != "a2" {
		t.Fatalf("head wrong: %v", []string{head[0].Info.ID(), head[3].Info.ID()})
	}
}

func TestSelectTail_NotEnoughTurns(t *testing.T) {
	history := []message.WithParts{user("u1"), assistant("a1", "u1")}
	head, tail := selectTail(history, 2)
	if head != nil || tail != "" {
		t.Fatalf("too-short history should yield no head, got head=%d tail=%q", len(head), tail)
	}
}

func TestSelectTail_ZeroTurnsDisabled(t *testing.T) {
	history := []message.WithParts{user("u1"), assistant("a1", "u1"), user("u2"), assistant("a2", "u2"), user("u3")}
	if head, tail := selectTail(history, 0); head != nil || tail != "" {
		t.Fatalf("tailTurns=0 disables compaction, got head=%d tail=%q", len(head), tail)
	}
}

func TestPrune_MarksOldToolOutputs(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const sid = "ses_prune"
	if _, err := db.Exec(`INSERT INTO project (id, worktree, time_created, time_updated) VALUES ('p','/tmp',0,0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session (id, project_id, slug, directory, version, time_created, time_updated)
		VALUES (?, 'p','s','/tmp','1',0,0)`, sid); err != nil {
		t.Fatal(err)
	}
	store := message.NewStore(db)
	ctx := context.Background()
	a := &message.AssistantMessage{ID: "msg_a", SessionID: sid, Role: message.RoleAssistant, ProviderID: "openai", ModelID: "gpt-4o"}
	if err := store.PutMessage(ctx, message.Info{Assistant: a}); err != nil {
		t.Fatal(err)
	}
	// 6 completed tool parts, ~15k tokens each (60k chars) -> 90k tokens total.
	big := strings.Repeat("x", 60_000)
	ids := []string{"prt_1", "prt_2", "prt_3", "prt_4", "prt_5", "prt_6"}
	for _, pid := range ids {
		st := message.ToolStateCompleted{Status: message.ToolCompleted, Input: map[string]any{}, Output: big, Title: "Bash", Metadata: map[string]any{}}
		st.Time.End = 1
		state, _ := json.Marshal(st)
		if err := store.PutPart(ctx, &message.ToolPart{
			PartBase: message.PartBase{ID: pid, SessionID: sid, MessageID: "msg_a"}, Type: "tool", CallID: pid, Tool: "bash", State: state}); err != nil {
			t.Fatal(err)
		}
	}

	eng := New(Config{Store: store})
	eng.prune(ctx, sid)

	parts, _ := store.Parts(ctx, "msg_a")
	var compacted, kept int
	for _, p := range parts {
		var st message.ToolStateCompleted
		_ = json.Unmarshal(p.(*message.ToolPart).State, &st)
		if st.Time.Compacted != nil {
			compacted++
		} else {
			kept++
		}
	}
	if compacted == 0 {
		t.Fatalf("expected some tool outputs pruned, got 0 (kept=%d)", kept)
	}
	if kept == 0 {
		t.Fatalf("prune must protect recent outputs, but pruned all")
	}
}

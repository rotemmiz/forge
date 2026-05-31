package mcp

import "testing"

func TestParseConfig(t *testing.T) {
	cfg := map[string]any{"mcp": map[string]any{
		"local-srv": map[string]any{
			"type":        "local",
			"command":     []any{"my-server", "--flag"},
			"environment": map[string]any{"KEY": "val"},
			"timeout":     float64(5000),
		},
		"remote-srv": map[string]any{
			"type":    "remote",
			"url":     "https://mcp.example.com",
			"headers": map[string]any{"Authorization": "Bearer x"},
			"enabled": false,
		},
		"bad": map[string]any{"type": "weird"}, // skipped
	}}
	got := ParseConfig(cfg)
	if len(got) != 2 {
		t.Fatalf("want 2 servers, got %d: %v", len(got), got)
	}
	loc := got["local-srv"]
	if loc.Type != "local" || len(loc.Command) != 2 || loc.Command[0] != "my-server" {
		t.Fatalf("local parse wrong: %+v", loc)
	}
	if loc.Environment["KEY"] != "val" || loc.Timeout != 5000 {
		t.Fatalf("local env/timeout wrong: %+v", loc)
	}
	if !loc.enabled() {
		t.Error("local should be enabled (absent ⇒ enabled)")
	}
	rem := got["remote-srv"]
	if rem.Type != "remote" || rem.URL != "https://mcp.example.com" || rem.Headers["Authorization"] == "" {
		t.Fatalf("remote parse wrong: %+v", rem)
	}
	if rem.enabled() {
		t.Error("remote should be disabled")
	}
}

func TestParseConfig_NoMCP(t *testing.T) {
	if ParseConfig(map[string]any{}) != nil {
		t.Fatal("no mcp key ⇒ nil")
	}
}

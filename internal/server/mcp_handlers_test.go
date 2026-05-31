package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/rotemmiz/forge/internal/auth"
)

func TestMCPStatus_Empty(t *testing.T) {
	h := newBackedServer(t, auth.Config{})
	rr, body := req(t, h, http.MethodGet, "/mcp", t.TempDir())
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if string(body) != "{}\n" && string(body) != "{}" {
		t.Fatalf("empty /mcp must be {}; got %q", body)
	}
}

func TestMCPStatus_DisabledFromConfig(t *testing.T) {
	h := newBackedServer(t, auth.Config{})
	dir := t.TempDir()
	// A disabled local server exercises the config→instance→manager→endpoint
	// path without spawning a process.
	cfg := `{"mcp":{"my-tool":{"type":"local","command":["my-mcp"],"enabled":false}}}`
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	rr, body := req(t, h, http.MethodGet, "/mcp", dir)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rr.Code, body)
	}
	var status map[string]struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatal(err)
	}
	if status["my-tool"].Status != "disabled" {
		t.Fatalf("my-tool status = %+v (body %s)", status["my-tool"], body)
	}
}

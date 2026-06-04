package server

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/rotemmiz/forge/internal/auth"
	"github.com/rotemmiz/forge/internal/bus"
	"github.com/rotemmiz/forge/internal/engine/catalog"
	"github.com/rotemmiz/forge/internal/engine/llm"
	"github.com/rotemmiz/forge/internal/engine/message"
	"github.com/rotemmiz/forge/internal/engine/registry"
	"github.com/rotemmiz/forge/internal/engine/tool"
	"github.com/rotemmiz/forge/internal/instance"
	"github.com/rotemmiz/forge/internal/push"
	"github.com/rotemmiz/forge/internal/session"
	"github.com/rotemmiz/forge/internal/storage"
)

// fullyWiredServer builds a daemon with every Options dependency populated so the
// complete set of real routes registers (the spec-driven 501 loop then fills the
// remaining reference operations). It backs the self-emit drift gate, which needs
// the maximal route table.
func fullyWiredServer(t *testing.T) http.Handler {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("OPENCODE_AUTH_CONTENT", "")

	db, err := storage.Open(filepath.Join(t.TempDir(), "forge.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	g := bus.NewGlobal()
	todos := tool.NewTodoStore()
	h, err := New(Options{
		Version:   "test",
		Auth:      auth.Config{},
		Cwd:       t.TempDir(),
		Sessions:  session.NewStore(db),
		Instances: instance.NewManager(g),
		Messages:  message.NewStore(db),
		Catalog:   catalog.Fixture(),
		Registry:  registry.New(tool.Bash{}, tool.Read{}, tool.Write{}, tool.Edit{}),
		Todos:     todos,
		Global:    g,
		Providers: func(context.Context, string, string) (llm.Provider, error) { return nil, nil },
		Push:      push.NewStore(db.DB),
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return h
}

package credstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// isolate points the store at a temp HOME so tests don't touch the real
// ~/.local/share/opencode/auth.json, and clears the env override.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_AUTH_CONTENT", "")
}

func TestSetGetRemove(t *testing.T) {
	isolate(t)
	if len(Load()) != 0 {
		t.Fatal("fresh store should be empty")
	}
	if err := Set("anthropic", json.RawMessage(`{"type":"api","key":"sk-1"}`)); err != nil {
		t.Fatal(err)
	}
	store := Load()
	rec, ok := store["anthropic"]
	if !ok {
		t.Fatalf("anthropic not stored: %v", store)
	}
	if TypeOf(rec) != "api" {
		t.Fatalf("type = %q", TypeOf(rec))
	}
	if err := Remove("anthropic"); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load()["anthropic"]; ok {
		t.Fatal("anthropic should be removed")
	}
}

func TestSet_NormalizesIDAndPersistsOthers(t *testing.T) {
	isolate(t)
	_ = Set("groq", json.RawMessage(`{"type":"api","key":"g"}`))
	_ = Set("openai/", json.RawMessage(`{"type":"api","key":"o"}`)) // trailing slash normalized
	store := Load()
	if _, ok := store["openai"]; !ok {
		t.Fatalf("trailing slash not normalized: %v", store)
	}
	if _, ok := store["groq"]; !ok {
		t.Fatal("setting a second provider dropped the first")
	}
}

func TestSet_FilePermissions(t *testing.T) {
	isolate(t)
	if err := Set("x", json.RawMessage(`{"type":"api","key":"k"}`)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(storePath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("auth.json mode = %o, want 600", perm)
	}
}

func TestSet_ConcurrentNoLostWrites(t *testing.T) {
	isolate(t)
	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("provider-%d", i)
			_ = Set(id, json.RawMessage(`{"type":"api","key":"k"}`))
		}(i)
	}
	wg.Wait()
	if got := len(Load()); got != n {
		t.Fatalf("concurrent writes lost: got %d of %d providers", got, n)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_AUTH_CONTENT", `{"google":{"type":"oauth","refresh":"r","access":"a","expires":1}}`)
	store := Load()
	if TypeOf(store["google"]) != "oauth" {
		t.Fatalf("env override not honored: %v", store)
	}
}

package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rotemmiz/forge/internal/auth"
	"github.com/rotemmiz/forge/internal/credstore"
	"github.com/rotemmiz/forge/internal/engine/catalog"
	"github.com/rotemmiz/forge/internal/resource"
)

func TestProviderAuthPutDelete(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_AUTH_CONTENT", "")
	h := newBackedServer(t, auth.Config{})

	// PUT an api key.
	put := func(body string) int {
		r := httptest.NewRequest(http.MethodPut, "/auth/anthropic", bytes.NewReader([]byte(body)))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, r)
		return rr.Code
	}
	if code := put(`{"type":"api","key":"sk-test"}`); code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	if credstore.TypeOf(credstore.Load()["anthropic"]) != "api" {
		t.Fatal("credential not persisted to the shared store")
	}

	// Invalid record → 400, store unchanged.
	if code := put(`{"type":"bogus"}`); code != http.StatusBadRequest {
		t.Fatalf("invalid type status = %d (want 400)", code)
	}

	// DELETE removes it.
	r := httptest.NewRequest(http.MethodDelete, "/auth/anthropic", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d", rr.Code)
	}
	if _, ok := credstore.Load()["anthropic"]; ok {
		t.Fatal("credential not removed")
	}
}

// TestProviderAuthFlipsConnected proves PUT /auth makes /provider report the
// provider connected (the end-to-end point of the credential store).
func TestProviderAuthFlipsConnected(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	t.Setenv("OPENCODE_AUTH_CONTENT", "")
	if err := credstore.Set("anthropic", []byte(`{"type":"api","key":"k"}`)); err != nil {
		t.Fatal(err)
	}
	cat := catalog.Catalog{"anthropic": {ID: "anthropic", Models: map[string]catalog.Model{"m": {ID: "m"}}}}
	list := resource.BuildProviderList(cat, map[string]any{})
	connected := false
	for _, id := range list.Connected {
		if id == "anthropic" {
			connected = true
		}
	}
	if !connected {
		t.Fatalf("anthropic should be connected after storing a credential; connected=%v", list.Connected)
	}
}

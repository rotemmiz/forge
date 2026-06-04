package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"
)

// mockToken builds a non-expiring transport.Token with the given access token.
func mockToken(access string) transport.Token {
	return transport.Token{AccessToken: access, TokenType: "Bearer"}
}

// mockTokenExpiring builds a transport.Token expiring in expiresIn seconds.
func mockTokenExpiring(access string, expiresIn int64) transport.Token {
	return transport.Token{AccessToken: access, TokenType: "Bearer", ExpiresIn: expiresIn}
}

// TestAuthStore_RoundTrip persists an entry and reads it back, proving the
// mcp-auth.json shape round-trips (tokens + clientInfo + flow state).
func TestAuthStore_RoundTrip(t *testing.T) {
	isolateAuthStore(t)

	if err := mutateAuthEntry("srv", func(e *authEntry) {
		e.Tokens = &authTokens{AccessToken: "acc", RefreshToken: "ref", Scope: "s"}
		e.ClientInfo = &authClientInfo{ClientID: "cid", ClientSecret: "sec"}
		e.CodeVerifier = "verif"
		e.OAuthState = "state"
		e.ServerURL = "https://srv"
	}); err != nil {
		t.Fatal(err)
	}

	e, ok := getAuthEntry("srv")
	if !ok {
		t.Fatal("entry not found after write")
	}
	if e.Tokens == nil || e.Tokens.AccessToken != "acc" || e.Tokens.RefreshToken != "ref" {
		t.Fatalf("tokens round-trip wrong: %+v", e.Tokens)
	}
	if e.ClientInfo == nil || e.ClientInfo.ClientID != "cid" {
		t.Fatalf("clientInfo round-trip wrong: %+v", e.ClientInfo)
	}
	if e.CodeVerifier != "verif" || e.OAuthState != "state" || e.ServerURL != "https://srv" {
		t.Fatalf("flow state round-trip wrong: %+v", e)
	}

	if err := removeAuthEntry("srv"); err != nil {
		t.Fatal(err)
	}
	if _, ok := getAuthEntry("srv"); ok {
		t.Fatal("entry should be gone after remove")
	}
}

// TestPersistentTokenStore_URLScope proves a stored token is only returned for
// the matching server URL (McpAuth.getForUrl), so a URL change forces re-auth.
func TestPersistentTokenStore_URLScope(t *testing.T) {
	isolateAuthStore(t)
	ctx := context.Background()

	store := &persistentTokenStore{name: "srv", serverURL: "https://a"}
	tk := mockToken("https-a-token")
	if err := store.SaveToken(ctx, &tk); err != nil {
		t.Fatal(err)
	}

	// Same URL → returns the token.
	tok, err := store.GetToken(ctx)
	if err != nil || tok.AccessToken != "https-a-token" {
		t.Fatalf("same-url GetToken = %v, err=%v", tok, err)
	}

	// Different URL → ErrNoToken (so the transport re-triggers auth).
	other := &persistentTokenStore{name: "srv", serverURL: "https://b"}
	if _, err := other.GetToken(ctx); err == nil {
		t.Fatal("token stored for https://a must not be returned for https://b")
	}
}

// TestPersistentTokenStore_ExpiresAt proves ExpiresIn is converted to an absolute
// expiry on save and read back as ExpiresAt.
func TestPersistentTokenStore_ExpiresAt(t *testing.T) {
	isolateAuthStore(t)
	ctx := context.Background()
	store := &persistentTokenStore{name: "srv", serverURL: "https://a"}

	tk := mockTokenExpiring("tok", 3600)
	if err := store.SaveToken(ctx, &tk); err != nil {
		t.Fatal(err)
	}
	tok, err := store.GetToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tok.ExpiresAt.IsZero() || tok.ExpiresAt.Before(time.Now()) {
		t.Fatalf("expiry not in the future: %v", tok.ExpiresAt)
	}
}

// refreshMock is a mock OAuth authorization server exposing just the RFC 8414
// metadata + token endpoints, with a token endpoint that accepts the
// refresh_token grant. It lets the MCP at-request refresh path be exercised
// without a live OAuth server.
type refreshMock struct {
	srv          *httptest.Server
	lastGrant    string
	lastRefresh  string
	rotateToken  string // refresh_token to return ("" = omit, exercising retain-old)
	newAccess    string
	refreshFails bool
}

func newRefreshMock(t *testing.T) *refreshMock {
	t.Helper()
	m := &refreshMock{newAccess: "refreshed-access", rotateToken: "rotated-refresh"}
	mux := http.NewServeMux()
	m.srv = httptest.NewServer(mux)
	base := m.srv.URL

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONResp(w, map[string]any{
			"issuer":                                base,
			"authorization_endpoint":                base + "/authorize",
			"token_endpoint":                        base + "/token",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"token_endpoint_auth_methods_supported": []string{"none"},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		m.lastGrant = r.Form.Get("grant_type")
		m.lastRefresh = r.Form.Get("refresh_token")
		if m.refreshFails {
			w.WriteHeader(http.StatusBadRequest)
			writeJSONResp(w, map[string]any{"error": "invalid_grant"})
			return
		}
		resp := map[string]any{
			"access_token": m.newAccess,
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		if m.rotateToken != "" {
			resp["refresh_token"] = m.rotateToken
		}
		writeJSONResp(w, resp)
	})

	t.Cleanup(m.srv.Close)
	return m
}

// newRefreshHandler builds an mcp-go OAuthHandler wired to the mock auth server
// and our persistent token store, the same store the live MCP transport uses.
func newRefreshHandler(m *refreshMock, store transport.TokenStore) *transport.OAuthHandler {
	h := transport.NewOAuthHandler(transport.OAuthConfig{
		ClientID:              "client-x",
		RedirectURI:           "http://127.0.0.1:19876/mcp/oauth/callback",
		TokenStore:            store,
		PKCEEnabled:           true,
		AuthServerMetadataURL: m.srv.URL + "/.well-known/oauth-authorization-server",
	})
	h.SetBaseURL(m.srv.URL)
	return h
}

// TestMCPRefresh_PersistsToStore proves the at-request refresh_token grant
// renews the access token and persists it back to mcp-auth.json via the
// persistent token store. mcp-go's getValidToken drives this on every request
// once a token is expired; here we invoke the public RefreshToken wrapper to
// assert the persisted shape deterministically.
func TestMCPRefresh_PersistsToStore(t *testing.T) {
	isolateAuthStore(t)
	ctx := context.Background()
	const serverURL = "https://mcp.example"
	store := &persistentTokenStore{name: "srv", serverURL: serverURL}

	// Seed an expired token with a refresh token (as a prior auth would leave it).
	if err := mutateAuthEntry("srv", func(e *authEntry) {
		e.Tokens = &authTokens{
			AccessToken:  "stale-access",
			RefreshToken: "old-refresh",
			ExpiresAt:    float64(time.Now().Add(-time.Minute).Unix()),
		}
		e.ServerURL = serverURL
	}); err != nil {
		t.Fatal(err)
	}

	m := newRefreshMock(t)
	h := newRefreshHandler(m, store)

	tok, err := h.RefreshToken(ctx, "old-refresh")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if tok.AccessToken != "refreshed-access" {
		t.Fatalf("refreshed access = %q", tok.AccessToken)
	}
	if m.lastGrant != "refresh_token" || m.lastRefresh != "old-refresh" {
		t.Fatalf("grant=%q refresh=%q, want refresh_token/old-refresh", m.lastGrant, m.lastRefresh)
	}

	// The renewed token must be persisted to mcp-auth.json in opencode shape.
	e, ok := getAuthEntry("srv")
	if !ok || e.Tokens == nil {
		t.Fatal("entry not persisted after refresh")
	}
	if e.Tokens.AccessToken != "refreshed-access" {
		t.Fatalf("persisted access = %q, want refreshed-access", e.Tokens.AccessToken)
	}
	if e.Tokens.RefreshToken != "rotated-refresh" {
		t.Fatalf("persisted refresh = %q, want rotated-refresh", e.Tokens.RefreshToken)
	}
	if e.Tokens.ExpiresAt <= float64(time.Now().Unix()) {
		t.Fatalf("expiresAt not advanced: %v", e.Tokens.ExpiresAt)
	}
	if e.ServerURL != serverURL {
		t.Fatalf("serverURL scope lost: %q", e.ServerURL)
	}
}

// TestMCPRefresh_RetainsRefreshTokenWhenNotRotated proves a refresh response
// that omits refresh_token keeps the old one persisted (mcp-go oauth.go:321-323).
func TestMCPRefresh_RetainsRefreshTokenWhenNotRotated(t *testing.T) {
	isolateAuthStore(t)
	ctx := context.Background()
	const serverURL = "https://mcp.example"
	store := &persistentTokenStore{name: "srv", serverURL: serverURL}
	if err := mutateAuthEntry("srv", func(e *authEntry) {
		e.Tokens = &authTokens{AccessToken: "stale", RefreshToken: "keep-me"}
		e.ServerURL = serverURL
	}); err != nil {
		t.Fatal(err)
	}

	m := newRefreshMock(t)
	m.rotateToken = "" // server does not rotate the refresh token
	h := newRefreshHandler(m, store)

	if _, err := h.RefreshToken(ctx, "keep-me"); err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	e, _ := getAuthEntry("srv")
	if e.Tokens == nil || e.Tokens.RefreshToken != "keep-me" {
		t.Fatalf("refresh token not retained: %+v", e.Tokens)
	}
	if e.Tokens.AccessToken != "refreshed-access" {
		t.Fatalf("access not updated: %q", e.Tokens.AccessToken)
	}
}

// TestMCPRefresh_FailureSurfacesError proves a rejected refresh returns an error
// (so the transport falls back to the re-auth/needs_auth path) and does NOT
// clobber the stored token.
func TestMCPRefresh_FailureSurfacesError(t *testing.T) {
	isolateAuthStore(t)
	ctx := context.Background()
	const serverURL = "https://mcp.example"
	store := &persistentTokenStore{name: "srv", serverURL: serverURL}
	if err := mutateAuthEntry("srv", func(e *authEntry) {
		e.Tokens = &authTokens{AccessToken: "stale", RefreshToken: "revoked"}
		e.ServerURL = serverURL
	}); err != nil {
		t.Fatal(err)
	}

	m := newRefreshMock(t)
	m.refreshFails = true
	h := newRefreshHandler(m, store)

	if _, err := h.RefreshToken(ctx, "revoked"); err == nil {
		t.Fatal("expected refresh failure error")
	}
	// Stored token must be untouched so the existing token / re-auth path stands.
	e, ok := getAuthEntry("srv")
	if !ok || e.Tokens == nil || e.Tokens.AccessToken != "stale" || e.Tokens.RefreshToken != "revoked" {
		t.Fatalf("stored token clobbered on refresh failure: %+v", e.Tokens)
	}
}

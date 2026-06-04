package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"
)

// writeOAuthRecord seeds auth.json with an oauth record for providerID via the
// same credstore path production uses.
func writeOAuthRecord(t *testing.T, providerID string, rec oauthRecord) {
	t.Helper()
	if err := persistOAuth(providerID, Token{
		Access:    rec.Access,
		Refresh:   rec.Refresh,
		Expires:   rec.Expires,
		AccountID: rec.AccountID,
	}); err != nil {
		t.Fatal(err)
	}
}

// jwtWithExp builds an unsigned JWT (header.payload.sig) whose exp claim is at
// expUnix seconds, for testing the JWT-exp proactive-refresh path.
func jwtWithExp(t *testing.T, expUnix int64) string {
	t.Helper()
	enc := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	header := enc(map[string]string{"alg": "none", "typ": "JWT"})
	payload := enc(map[string]any{"exp": expUnix})
	return header + "." + payload + ".sig"
}

// readPersisted reads back providerID's persisted oauth record from auth.json.
func readPersisted(t *testing.T, authPath, providerID string) oauthRecord {
	t.Helper()
	raw, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("auth.json not written: %v", err)
	}
	var store map[string]oauthRecord
	if err := json.Unmarshal(raw, &store); err != nil {
		t.Fatal(err)
	}
	return store[providerID]
}

func TestAccessUnknownProvider(t *testing.T) {
	setAuthDir(t)
	s := newTestXaiService(t, "http://unused", "https://auth.example/authorize")
	if _, err := s.Access(context.Background(), "nope"); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("err = %v, want ErrUnknownProvider", err)
	}
}

func TestAccessNoStoredToken(t *testing.T) {
	setAuthDir(t)
	s := newTestXaiService(t, "http://unused", "https://auth.example/authorize")
	if _, err := s.Access(context.Background(), "xai"); !errors.Is(err, ErrNoOAuthToken) {
		t.Fatalf("err = %v, want ErrNoOAuthToken", err)
	}
}

// TestAccessFreshTokenNoRefresh: a token comfortably beyond the skew window is
// returned as-is, with no call to the token endpoint.
func TestAccessFreshTokenNoRefresh(t *testing.T) {
	setAuthDir(t)
	called := false
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	// Opaque access token, expires 1h out — well past the 2min skew.
	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  "fresh-access",
		Refresh: "ref",
		Expires: time.Now().Add(time.Hour).UnixMilli(),
	})

	got, err := s.Access(context.Background(), "xai")
	if err != nil {
		t.Fatal(err)
	}
	if got != "fresh-access" {
		t.Fatalf("access = %q, want fresh-access", got)
	}
	if called {
		t.Fatal("token endpoint called for a fresh token")
	}
}

// TestAccessRefreshesExpiredToken: an expired stored token triggers a
// refresh_token grant; the new token is returned AND persisted in opencode shape.
func TestAccessRefreshesExpiredToken(t *testing.T) {
	authPath := setAuthDir(t)
	var gotForm url.Values
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	writeOAuthRecord(t, "xai", oauthRecord{
		Access:    "stale-access",
		Refresh:   "old-refresh",
		Expires:   time.Now().Add(-time.Minute).UnixMilli(), // already expired
		AccountID: "acct-42",
	})

	got, err := s.Access(context.Background(), "xai")
	if err != nil {
		t.Fatal(err)
	}
	if got != "new-access" {
		t.Fatalf("access = %q, want new-access", got)
	}
	// The refresh_token grant must have been sent with the stored refresh token.
	if gotForm.Get("grant_type") != "refresh_token" {
		t.Fatalf("grant_type = %q, want refresh_token", gotForm.Get("grant_type"))
	}
	if gotForm.Get("refresh_token") != "old-refresh" {
		t.Fatalf("refresh_token = %q, want old-refresh", gotForm.Get("refresh_token"))
	}
	// Persisted record reflects the rotated token + new expiry + retained account.
	rec := readPersisted(t, authPath, "xai")
	if rec.Type != "oauth" || rec.Access != "new-access" || rec.Refresh != "new-refresh" {
		t.Fatalf("bad persisted record: %+v", rec)
	}
	if rec.AccountID != "acct-42" {
		t.Fatalf("accountId not carried over: %q", rec.AccountID)
	}
	if rec.Expires <= time.Now().UnixMilli() {
		t.Fatalf("expires not advanced: %d", rec.Expires)
	}
}

// TestAccessRetainsRefreshTokenWhenNotRotated: a refresh response that omits a
// new refresh_token keeps the old one persisted (xai.ts:614 parity).
func TestAccessRetainsRefreshTokenWhenNotRotated(t *testing.T) {
	authPath := setAuthDir(t)
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"rotated-access","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  "stale",
		Refresh: "keep-me",
		Expires: 0, // unknown expiry → treated as needs-refresh
	})

	got, err := s.Access(context.Background(), "xai")
	if err != nil {
		t.Fatal(err)
	}
	if got != "rotated-access" {
		t.Fatalf("access = %q", got)
	}
	rec := readPersisted(t, authPath, "xai")
	if rec.Refresh != "keep-me" {
		t.Fatalf("refresh token not retained: %q", rec.Refresh)
	}
}

// TestAccessRefreshesExpiringJWT: an opaque stored expiry that looks healthy but
// a JWT access token whose exp is within skew still triggers a refresh.
func TestAccessRefreshesExpiringJWT(t *testing.T) {
	setAuthDir(t)
	called := false
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"jwt-refreshed","refresh_token":"r2","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	// Stored expires is far out (would NOT trigger refresh alone), but the JWT exp
	// is 30s away — inside the 2min skew — so the JWT check forces the refresh.
	jwt := jwtWithExp(t, time.Now().Add(30*time.Second).Unix())
	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  jwt,
		Refresh: "r1",
		Expires: time.Now().Add(time.Hour).UnixMilli(),
	})

	got, err := s.Access(context.Background(), "xai")
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected refresh for expiring JWT")
	}
	if got != "jwt-refreshed" {
		t.Fatalf("access = %q", got)
	}
}

// TestAccessHealthyJWTNoRefresh: a JWT with a far-future exp and a healthy stored
// expiry is used as-is.
func TestAccessHealthyJWTNoRefresh(t *testing.T) {
	setAuthDir(t)
	called := false
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	jwt := jwtWithExp(t, time.Now().Add(time.Hour).Unix())
	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  jwt,
		Refresh: "r1",
		Expires: time.Now().Add(time.Hour).UnixMilli(),
	})

	got, err := s.Access(context.Background(), "xai")
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("token endpoint called for a healthy JWT")
	}
	if got != jwt {
		t.Fatalf("access mismatch")
	}
}

// TestAccessRefreshFailureNeedsReauth: a refresh that the auth server rejects
// surfaces ErrNeedsReauth and does NOT clobber the stored record.
func TestAccessRefreshFailureNeedsReauth(t *testing.T) {
	authPath := setAuthDir(t)
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  "stale",
		Refresh: "revoked-refresh",
		Expires: time.Now().Add(-time.Minute).UnixMilli(),
	})

	_, err := s.Access(context.Background(), "xai")
	if !errors.Is(err, ErrNeedsReauth) {
		t.Fatalf("err = %v, want ErrNeedsReauth", err)
	}
	// The stored record must be untouched so the user can re-authorize.
	rec := readPersisted(t, authPath, "xai")
	if rec.Access != "stale" || rec.Refresh != "revoked-refresh" {
		t.Fatalf("stored record clobbered on refresh failure: %+v", rec)
	}
}

// TestAccessExpiredNoRefreshToken: an expired token with no refresh token cannot
// be renewed, so Access reports ErrNeedsReauth without hitting the network.
func TestAccessExpiredNoRefreshToken(t *testing.T) {
	setAuthDir(t)
	called := false
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  "stale",
		Refresh: "",
		Expires: time.Now().Add(-time.Minute).UnixMilli(),
	})

	if _, err := s.Access(context.Background(), "xai"); !errors.Is(err, ErrNeedsReauth) {
		t.Fatalf("err = %v, want ErrNeedsReauth", err)
	}
	if called {
		t.Fatal("token endpoint should not be called without a refresh token")
	}
}

// TestAccessConcurrentRefreshSingleFlight: many concurrent Access calls on an
// expired token collapse onto ONE refresh_token round-trip (so a rotating
// refresh token isn't replayed), and all observe the same new access token.
func TestAccessConcurrentRefreshSingleFlight(t *testing.T) {
	setAuthDir(t)
	var mu sync.Mutex
	var hits int
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		// Hold the request briefly so concurrent callers pile up on the flight.
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"sf-access","refresh_token":"sf-refresh","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	s := newTestXaiService(t, tokenSrv.URL, "https://auth.example/authorize")

	writeOAuthRecord(t, "xai", oauthRecord{
		Access:  "stale",
		Refresh: "old",
		Expires: time.Now().Add(-time.Minute).UnixMilli(),
	})

	const n = 16
	var wg sync.WaitGroup
	errs := make([]error, n)
	got := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i], errs[i] = s.Access(context.Background(), "xai")
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got[i] != "sf-access" {
			t.Fatalf("call %d access = %q", i, got[i])
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if hits != 1 {
		t.Fatalf("token endpoint hit %d times, want 1 (single-flight)", hits)
	}
}

// TestNeedsRefreshBoundary checks the skew boundary directly.
func TestNeedsRefreshBoundary(t *testing.T) {
	fixed := time.Unix(1_700_000_000, 0)
	now = func() time.Time { return fixed }
	t.Cleanup(func() { now = time.Now })

	cases := []struct {
		name    string
		expires int64
		want    bool
	}{
		{"zero expiry", 0, true},
		{"already expired", fixed.Add(-time.Second).UnixMilli(), true},
		{"inside skew", fixed.Add(accessTokenRefreshSkew - time.Second).UnixMilli(), true},
		{"just outside skew", fixed.Add(accessTokenRefreshSkew + time.Second).UnixMilli(), false},
		{"far future", fixed.Add(time.Hour).UnixMilli(), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Opaque access token so only the stored-expiry path is exercised.
			if got := needsRefresh(oauthRecord{Access: "opaque", Expires: c.expires}); got != c.want {
				t.Fatalf("needsRefresh(expires=%d) = %v, want %v", c.expires, got, c.want)
			}
		})
	}
}

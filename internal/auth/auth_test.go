package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestNoPasswordIsPassthrough(t *testing.T) {
	c := Config{Username: "opencode", Password: ""}
	if c.Required() {
		t.Fatal("Required() should be false with empty password")
	}
	rr := httptest.NewRecorder()
	c.Middleware(okHandler()).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/config", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (auth disabled)", rr.Code)
	}
}

func TestMissingCredentials401(t *testing.T) {
	c := Config{Username: "opencode", Password: "secret"}
	rr := httptest.NewRecorder()
	c.Middleware(okHandler()).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/config", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != wwwAuthenticate {
		t.Errorf("WWW-Authenticate = %q, want %q", got, wwwAuthenticate)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("401 body = %q, want empty", rr.Body.String())
	}
}

func TestBasicHeaderOK(t *testing.T) {
	c := Config{Username: "opencode", Password: "secret"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	req.SetBasicAuth("opencode", "secret")
	c.Middleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestAuthTokenQueryOK(t *testing.T) {
	c := Config{Username: "opencode", Password: "secret"}
	token := base64.StdEncoding.EncodeToString([]byte("opencode:secret"))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config?auth_token="+token, nil)
	c.Middleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (auth_token query)", rr.Code)
	}
}

func TestWrongPassword401(t *testing.T) {
	c := Config{Username: "opencode", Password: "secret"}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	req.SetBasicAuth("opencode", "wrong")
	c.Middleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthTokenTakesPrecedenceOverBadHeader(t *testing.T) {
	c := Config{Username: "opencode", Password: "secret"}
	token := base64.StdEncoding.EncodeToString([]byte("opencode:secret"))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config?auth_token="+token, nil)
	req.Header.Set("Authorization", "Basic garbage")
	c.Middleware(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (auth_token wins)", rr.Code)
	}
}

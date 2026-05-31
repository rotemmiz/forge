package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rotemmiz/forge/internal/credstore"
)

// registerProviderAuthRoutes wires the credential write/delete endpoints onto
// opencode's shared auth.json store (auth/index.ts). PUT sets a provider's
// credential, DELETE removes it; both are reflected in /provider's connected
// list. The OAuth authorize/callback flow and GET /provider/auth method listing
// are deferred (logged in known-divergences).
func registerProviderAuthRoutes(reg func(method, path string, h http.HandlerFunc)) {
	reg(http.MethodPut, "/auth/{providerID}", putAuthHandler())
	reg(http.MethodDelete, "/auth/{providerID}", deleteAuthHandler())
}

// validAuthTypes are the credential record kinds opencode's Auth union accepts.
var validAuthTypes = map[string]bool{"api": true, "oauth": true, "wellknown": true}

// putAuthHandler stores a provider credential (Auth: api|oauth|wellknown).
func putAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerID := chi.URLParam(r, "providerID")
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "BadRequest", "could not read body")
			return
		}
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(body, &probe); err != nil || !validAuthTypes[probe.Type] {
			writeError(w, http.StatusBadRequest, "BadRequest",
				"auth record must have type api, oauth, or wellknown")
			return
		}
		if err := credstore.Set(providerID, json.RawMessage(body)); err != nil {
			writeError(w, http.StatusInternalServerError, "AuthStoreError", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, true)
	}
}

// deleteAuthHandler removes a provider credential.
func deleteAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerID := chi.URLParam(r, "providerID")
		if err := credstore.Remove(providerID); err != nil {
			writeError(w, http.StatusInternalServerError, "AuthStoreError", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, true)
	}
}

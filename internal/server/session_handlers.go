package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rotemmiz/forge/internal/session"
)

// registerSessionRoutes wires the M2 session CRUD endpoints. Paths use the spec
// param name {sessionID}.
func registerSessionRoutes(reg func(method, path string, h http.HandlerFunc), store *session.Store) {
	reg(http.MethodPost, "/session", createSession(store))
	reg(http.MethodGet, "/session", listSessions(store))
	reg(http.MethodGet, "/session/{sessionID}", getSession(store))
	reg(http.MethodDelete, "/session/{sessionID}", deleteSession(store))
	reg(http.MethodPost, "/session/{sessionID}/fork", forkSession(store))
	reg(http.MethodGet, "/session/{sessionID}/children", childrenSession(store))
}

func createSession(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info, err := store.Create(r.Context(), DirectoryFromContext(r.Context()))
		if err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func listSessions(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := store.List(r.Context())
		if err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getSession(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := chi.URLParam(r, "sessionID")
		info, err := store.Get(r.Context(), sid)
		if errors.Is(err, session.ErrNotFound) {
			writeSessionNotFound(w, sid)
			return
		}
		if err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func deleteSession(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := chi.URLParam(r, "sessionID")
		ok, err := store.Delete(r.Context(), sid)
		if err != nil {
			writeInternal(w, err)
			return
		}
		if !ok {
			writeSessionNotFound(w, sid)
			return
		}
		// opencode returns a bare boolean true on a successful delete.
		writeJSON(w, http.StatusOK, true)
	}
}

func forkSession(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := chi.URLParam(r, "sessionID")
		info, err := store.Fork(r.Context(), sid)
		if errors.Is(err, session.ErrNotFound) {
			writeSessionNotFound(w, sid)
			return
		}
		if err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func childrenSession(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid := chi.URLParam(r, "sessionID")
		// requireSession first: a missing parent 404s before listing children
		// (session.ts:86-88).
		if _, err := store.Get(r.Context(), sid); errors.Is(err, session.ErrNotFound) {
			writeSessionNotFound(w, sid)
			return
		} else if err != nil {
			writeInternal(w, err)
			return
		}
		list, err := store.Children(r.Context(), sid)
		if err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	}
}

// writeSessionNotFound emits opencode's 404 envelope:
// {"data":{"message":"Session not found: <id>"},"name":"NotFoundError"}.
func writeSessionNotFound(w http.ResponseWriter, sessionID string) {
	writeJSON(w, http.StatusNotFound, map[string]any{
		"name": "NotFoundError",
		"data": map[string]any{"message": "Session not found: " + sessionID},
	})
}

func writeInternal(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"_tag": "InternalError", "message": err.Error(),
	})
}

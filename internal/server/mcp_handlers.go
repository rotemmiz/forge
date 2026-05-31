package server

import (
	"net/http"

	"github.com/rotemmiz/forge/internal/instance"
	"github.com/rotemmiz/forge/internal/mcp"
)

// registerMCPRoutes wires GET /mcp (server status). The mutating endpoints
// (add/connect/auth) and OAuth are deferred (logged in known-divergences).
func registerMCPRoutes(reg func(method, path string, h http.HandlerFunc), instances *instance.Manager) {
	reg(http.MethodGet, "/mcp", mcpStatusHandler(instances))
}

// mcpStatusHandler returns each configured MCP server's status for the request
// directory (connecting enabled local servers on first access).
func mcpStatusHandler(instances *instance.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst := instances.Get(DirectoryFromContext(r.Context()))
		status := inst.MCP.Status(r.Context())
		if status == nil {
			status = map[string]mcp.Status{}
		}
		writeJSON(w, http.StatusOK, status)
	}
}

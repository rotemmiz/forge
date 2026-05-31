package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// defaultTimeout is opencode's MCP request timeout (mcp/index.ts:37, 30s — the
// config description says 5s but the code uses 30s).
const defaultTimeout = 30 * time.Second

// Status is a server's connection status (openapi MCPStatus). Error is set only
// for "failed".
type Status struct {
	Status string `json:"status"` // connected | disabled | failed
	Error  string `json:"error,omitempty"`
}

// conn is the subset of mcp-go's client used here (so tests can substitute an
// in-process client).
type conn interface {
	Start(ctx context.Context) error
	Initialize(ctx context.Context, req mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, req mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Close() error
}

// Manager owns one instance's MCP clients. Connection is lazy (on first Status/
// Tools) so creating an instance stays cheap; results are cached for the
// instance's lifetime.
type Manager struct {
	servers map[string]Server
	dial    func(ctx context.Context, s Server) (conn, error)

	once    sync.Once
	mu      sync.Mutex
	closed  bool
	status  map[string]Status
	clients map[string]conn
	tools   map[string][]mcp.Tool // server name → its tools
}

// NewManager builds a manager for the given server configs.
func NewManager(servers map[string]Server) *Manager {
	return &Manager{servers: servers, dial: stdioDial}
}

// Status connects (once) and returns each server's status.
func (m *Manager) Status(ctx context.Context) map[string]Status {
	m.connect(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]Status, len(m.status))
	for k, v := range m.status {
		out[k] = v
	}
	return out
}

// connect dials every enabled server once, caching status and tool lists. It
// dials WITHOUT holding m.mu (dialing spawns processes / does I/O), then
// publishes the results under m.mu — so a concurrent Close (which also takes
// m.mu) can't race the map writes. If Close ran during the dial, the freshly
// dialed clients are closed and nothing is published.
func (m *Manager) connect(ctx context.Context) {
	m.once.Do(func() {
		m.mu.Lock()
		closed := m.closed
		servers := m.servers
		m.mu.Unlock()
		if closed {
			return
		}

		status := map[string]Status{}
		clients := map[string]conn{}
		tools := map[string][]mcp.Tool{}
		for name, s := range servers {
			if !s.enabled() {
				status[name] = Status{Status: "disabled"}
				continue
			}
			c, tl, err := m.dialAndList(ctx, s)
			if err != nil {
				status[name] = Status{Status: "failed", Error: errString(err)}
				continue
			}
			clients[name] = c
			tools[name] = tl
			status[name] = Status{Status: "connected"}
		}

		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			for _, c := range clients {
				_ = c.Close()
			}
			return
		}
		m.status, m.clients, m.tools = status, clients, tools
		m.mu.Unlock()
	})
}

// errString returns a non-empty error message (the failed status requires one).
func errString(err error) string {
	if s := err.Error(); s != "" {
		return s
	}
	return "connection failed"
}

// dialAndList connects, initializes, and lists a server's tools.
func (m *Manager) dialAndList(ctx context.Context, s Server) (conn, []mcp.Tool, error) {
	timeout := defaultTimeout
	if s.Timeout > 0 {
		timeout = time.Duration(s.Timeout) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	c, err := m.dial(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	// Note: the stdio constructor already spawns the subprocess (under its own
	// background ctx), so this Start is idempotent and the timeout ctx bounds
	// Initialize/ListTools rather than the spawn itself.
	if err := c.Start(ctx); err != nil {
		_ = c.Close()
		return nil, nil, err
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "forge", Version: "0.0.1"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		_ = c.Close()
		return nil, nil, err
	}
	res, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		_ = c.Close()
		return nil, nil, err
	}
	return c, res.Tools, nil
}

// Close shuts down all connected clients and marks the manager closed so an
// in-flight connect won't publish (or leak) new clients.
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	clients := m.clients
	m.clients = nil
	m.mu.Unlock()
	for _, c := range clients {
		_ = c.Close()
	}
}

// stdioDial is the default transport: local servers spawn a subprocess; remote
// servers are not yet supported (HTTP/SSE transports + OAuth are a follow-up).
func stdioDial(_ context.Context, s Server) (conn, error) {
	if s.Type != "local" {
		return nil, fmt.Errorf("remote MCP servers are not yet supported")
	}
	if len(s.Command) == 0 {
		return nil, fmt.Errorf("local MCP server has no command")
	}
	env := make([]string, 0, len(s.Environment))
	for k, v := range s.Environment {
		env = append(env, k+"="+v)
	}
	c, err := mcpgo.NewStdioMCPClient(s.Command[0], env, s.Command[1:]...)
	if err != nil {
		return nil, err
	}
	return c, nil
}

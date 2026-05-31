package instance

import (
	"sync"

	"github.com/rotemmiz/forge/internal/bus"
	"github.com/rotemmiz/forge/internal/config"
	"github.com/rotemmiz/forge/internal/engine/permission"
	"github.com/rotemmiz/forge/internal/engine/question"
	"github.com/rotemmiz/forge/internal/engine/runstate"
	"github.com/rotemmiz/forge/internal/mcp"
	"github.com/rotemmiz/forge/internal/pty"
)

// Context is the per-directory in-memory state for one project instance. It
// holds the instance event bus, PTY manager, and the agent engine's
// per-instance state (the permission manager and the per-session run lock);
// config/LSP attach here in later milestones (plan 01 §7).
type Context struct {
	Directory   string
	Bus         *bus.Bus
	Pty         *pty.Manager
	Permissions *permission.Manager
	Questions   *question.Manager
	RunState    *runstate.RunState
	// MCP holds this instance's configured MCP servers (connection is lazy).
	MCP *mcp.Manager
}

// Manager is the directory→instance cache. Instances are created on first use
// and kept for the server lifetime (opencode keeps them with no TTL;
// project/instance-store.ts:105-120). The cache is keyed by the canonical
// (symlink-resolved) directory path produced by directory resolution.
type Manager struct {
	mu        sync.Mutex
	instances map[string]*Context
	global    *bus.Global
}

// NewManager creates an empty instance manager whose instance buses forward to
// the given global bus.
func NewManager(global *bus.Global) *Manager {
	return &Manager{instances: make(map[string]*Context), global: global}
}

// DisposeAll tears down every instance on shutdown: it emits
// server.instance.disposed on each instance bus (which terminates that
// instance's /event SSE stream — handlers/event.ts:30-31) and shuts down its
// PTY sessions, then clears the cache (project/instance-store.ts:77-89).
func (m *Manager) DisposeAll() {
	m.mu.Lock()
	contexts := make([]*Context, 0, len(m.instances))
	for _, c := range m.instances {
		contexts = append(contexts, c)
	}
	m.instances = make(map[string]*Context)
	m.mu.Unlock()

	for _, c := range contexts {
		c.Bus.Publish(bus.NewEvent(bus.EventInstanceDisposed, map[string]any{"directory": c.Directory}))
		if c.Pty != nil {
			c.Pty.Shutdown()
		}
		if c.MCP != nil {
			c.MCP.Close()
		}
	}
}

// Get returns the instance for directory, creating it on first use. Creation is
// trivial today (a fresh bus + PTY manager), so a single lock suffices; when
// init grows expensive (config/LSP) this becomes the single-flight point
// (plan 01 §7).
func (m *Manager) Get(directory string) *Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.instances[directory]; ok {
		return c
	}
	instBus := bus.NewInstanceBus(directory, m.global)
	c := &Context{
		Directory: directory,
		Bus:       instBus,
		// configShell is wired from config in a later milestone; PreferredShell
		// falls back to $SHELL / a platform default until then.
		Pty:         pty.NewManager(directory, ""),
		Permissions: permission.NewManager(instBus),
		Questions:   question.NewManager(instBus),
		RunState:    runstate.New(),
		MCP:         mcp.NewManager(mcpServers(directory)),
	}
	m.instances[directory] = c
	return c
}

// mcpServers loads the directory's MCP server configs (empty on error, so a bad
// config never blocks instance creation).
func mcpServers(directory string) map[string]mcp.Server {
	cfg, err := config.Load(directory)
	if err != nil {
		return nil
	}
	return mcp.ParseConfig(cfg)
}

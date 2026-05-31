package mcp

import (
	"context"
	"errors"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// inProcessServer builds an in-process MCP server exposing one tool.
func inProcessServer() *server.MCPServer {
	s := server.NewMCPServer("test", "1.0.0")
	s.AddTool(
		mcp.NewTool("ping", mcp.WithDescription("returns pong")),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("pong"), nil
		},
	)
	return s
}

func TestManager_ConnectsAndListsTools(t *testing.T) {
	m := NewManager(map[string]Server{
		"echo": {Type: "local", Command: []string{"unused"}},
		"off":  {Type: "local", Command: []string{"unused"}, Enabled: boolPtr(false)},
	})
	m.dial = func(_ context.Context, _ Server) (conn, error) {
		return mcpgo.NewInProcessClient(inProcessServer())
	}

	status := m.Status(context.Background())
	if status["echo"].Status != "connected" {
		t.Fatalf("echo status = %+v", status["echo"])
	}
	if status["off"].Status != "disabled" {
		t.Fatalf("off status = %+v", status["off"])
	}
	m.mu.Lock()
	tools := m.tools["echo"]
	m.mu.Unlock()
	if len(tools) != 1 || tools[0].Name != "ping" {
		t.Fatalf("echo tools = %+v", tools)
	}
	m.Close()
}

func TestManager_DialFailureIsFailed(t *testing.T) {
	m := NewManager(map[string]Server{"broken": {Type: "local", Command: []string{"x"}}})
	m.dial = func(_ context.Context, _ Server) (conn, error) {
		return nil, errors.New("spawn failed")
	}
	status := m.Status(context.Background())
	if status["broken"].Status != "failed" || status["broken"].Error != "spawn failed" {
		t.Fatalf("broken status = %+v", status["broken"])
	}
}

func TestManager_CloseRacesConnect(t *testing.T) {
	// Close() must not race the lazy connect's map writes (regression for the
	// nil-map panic at shutdown). Run under -race.
	for i := 0; i < 50; i++ {
		m := NewManager(map[string]Server{"echo": {Type: "local", Command: []string{"x"}}})
		m.dial = func(_ context.Context, _ Server) (conn, error) {
			return mcpgo.NewInProcessClient(inProcessServer())
		}
		result := make(chan map[string]Status, 1)
		go func() { result <- m.Status(context.Background()) }()
		m.Close()
		if s := <-result; s == nil {
			t.Fatal("Status returned a nil map")
		}
	}
}

func TestStdioDial_RemoteUnsupported(t *testing.T) {
	if _, err := stdioDial(context.Background(), Server{Type: "remote", URL: "https://x"}); err == nil {
		t.Fatal("remote should be unsupported in this slice")
	}
}

func boolPtr(b bool) *bool { return &b }

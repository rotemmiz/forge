package forgeclient

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rotemmiz/forge/sdk/go/gen"
)

// TestLive_Smoke exercises the generated REST client + SSE consumer against a
// real running daemon (opencode or Forge). SKIPPED unless FORGE_SDK_TEST_URL is
// set, so deterministic CI never needs a daemon:
//
//	FORGE_SDK_TEST_URL  e.g. http://localhost:4096
//	FORGE_SDK_TEST_USER / FORGE_SDK_TEST_PASS  (optional Basic auth)
//	FORGE_SDK_TEST_DIR  (optional x-opencode-directory; defaults to cwd)
//
//	FORGE_SDK_TEST_URL=http://localhost:4096 go test ./sdk/go -run TestLive -v
func TestLive_Smoke(t *testing.T) {
	url := os.Getenv("FORGE_SDK_TEST_URL")
	if url == "" {
		t.Skip("set FORGE_SDK_TEST_URL to run the live SDK smoke test")
	}
	dir := os.Getenv("FORGE_SDK_TEST_DIR")
	if dir == "" {
		dir, _ = os.Getwd()
	}
	c, err := New(url, Options{
		Directory: dir,
		Username:  os.Getenv("FORGE_SDK_TEST_USER"),
		Password:  os.Getenv("FORGE_SDK_TEST_PASS"),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Health.
	if err := c.Health(ctx); err != nil {
		t.Fatalf("health: %v", err)
	}
	t.Logf("health ok against %s", url)

	// 2. Generated REST client: list sessions.
	resp, err := c.API.SessionListWithResponse(ctx, &gen.SessionListParams{})
	if err != nil {
		t.Fatalf("session list: %v", err)
	}
	t.Logf("session list status=%d", resp.StatusCode())

	// 3. SSE: connect and confirm the stream opens (server.connected arrives first).
	stream, err := c.GlobalEvents(ctx)
	if err != nil {
		t.Fatalf("global events: %v", err)
	}
	defer stream.Close()
	select {
	case ev := <-stream.Events():
		t.Logf("first SSE event: type=%s id=%s", ev.Type, ev.ID)
	case err := <-stream.Err():
		t.Fatalf("sse error before any event: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("no SSE event within 5s")
	}
}

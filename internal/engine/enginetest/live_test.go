package enginetest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rotemmiz/forge/internal/bus"
	"github.com/rotemmiz/forge/internal/engine"
	"github.com/rotemmiz/forge/internal/engine/catalog"
	"github.com/rotemmiz/forge/internal/engine/llm"
	"github.com/rotemmiz/forge/internal/engine/message"
	"github.com/rotemmiz/forge/internal/engine/permission"
	"github.com/rotemmiz/forge/internal/engine/provider/openai"
	"github.com/rotemmiz/forge/internal/engine/registry"
	"github.com/rotemmiz/forge/internal/engine/tool"
	"github.com/rotemmiz/forge/internal/storage"
)

// TestLive_RealTextPrompt drives the engine through the real OpenAI-compatible
// client against a free-tier provider. It is the "one real text prompt end-to-end"
// check and is SKIPPED unless the endpoint is configured, so deterministic CI
// never touches the network:
//
//	FORGE_TEST_BASE_URL  e.g. https://api.groq.com/openai/v1
//	FORGE_TEST_MODEL     e.g. llama-3.3-70b-versatile
//	FORGE_TEST_API_KEY   provider key (omit for keyless local endpoints, e.g. Ollama)
//
// Example:
//
//	FORGE_TEST_BASE_URL=https://api.groq.com/openai/v1 \
//	FORGE_TEST_MODEL=llama-3.3-70b-versatile FORGE_TEST_API_KEY=$GROQ_API_KEY \
//	go test ./internal/engine/enginetest -run TestLive -v
func TestLive_RealTextPrompt(t *testing.T) {
	baseURL := os.Getenv("FORGE_TEST_BASE_URL")
	model := os.Getenv("FORGE_TEST_MODEL")
	if baseURL == "" || model == "" {
		t.Skip("set FORGE_TEST_BASE_URL and FORGE_TEST_MODEL to run the live provider test")
	}
	apiKey := os.Getenv("FORGE_TEST_API_KEY")

	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	const sessionID = "ses_live"
	if _, err := db.Exec(`INSERT INTO project (id, worktree, time_created, time_updated) VALUES ('p','/tmp',0,0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session (id, project_id, slug, directory, version, time_created, time_updated)
		VALUES (?, 'p','s','/tmp','1',0,0)`, sessionID); err != nil {
		t.Fatal(err)
	}
	store := message.NewStore(db)
	b := bus.NewInstanceBus(sessionID, nil)

	eng := engine.New(engine.Config{
		Store: store, Catalog: catalog.Fixture(),
		Registry:    registry.New(tool.Bash{}, tool.Read{}),
		Permissions: permission.NewManager(b), Bus: b, Directory: t.TempDir(),
		Rulesets: []permission.Ruleset{{{Permission: "*", Pattern: "*", Action: permission.ActionAllow}}},
		Providers: func(context.Context, string, string) (llm.Provider, error) {
			return openai.New(openai.Options{BaseURL: baseURL, APIKey: apiKey, Model: model}), nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := eng.Prompt(ctx, engine.PromptInput{
		SessionID: sessionID, Provider: "openai-compatible", Model: model,
		Parts: []engine.PartInput{{Type: "text", Text: "Reply with exactly the word: pong"}},
	})
	if err != nil {
		t.Fatalf("live prompt: %v", err)
	}
	if out.Info.Assistant == nil || out.Info.Assistant.Finish == "" {
		t.Fatalf("assistant did not finish: %+v", out.Info.Assistant)
	}
	var text string
	for _, p := range out.Parts {
		if tp, ok := p.(*message.TextPart); ok {
			text += tp.Text
		}
	}
	if text == "" {
		t.Fatalf("live response had no text (finish=%s)", out.Info.Assistant.Finish)
	}
	t.Logf("live model replied: %q (finish=%s, tokens in=%v out=%v cost=$%.6f)",
		text, out.Info.Assistant.Finish, out.Info.Assistant.Tokens.Input, out.Info.Assistant.Tokens.Output, out.Info.Assistant.Cost)
}

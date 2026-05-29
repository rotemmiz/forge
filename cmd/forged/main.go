// Command forged is the Forge daemon: a ground-up, interop-first alternative
// to opencode that is wire-compatible with its HTTP+SSE+WebSocket API.
//
// It serves GET /global/health, GET /doc, GET /config (M1), and session CRUD
// (M2), with the opencode-compatible auth + directory middleware chain; every
// other documented operation still returns a structured 501 until the remaining
// plan-01 milestones land.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rotemmiz/forge/internal/auth"
	"github.com/rotemmiz/forge/internal/server"
	"github.com/rotemmiz/forge/internal/session"
	"github.com/rotemmiz/forge/internal/storage"
)

// version is the daemon version, overridable at build time via
// -ldflags "-X main.version=...".
var version = "0.0.1"

func main() {
	port := flag.Int("port", 4096, "HTTP listen port (0 = OS-assigned)")
	host := flag.String("host", "127.0.0.1", "host/interface to bind")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if err := run(*host, *port); err != nil {
		log.Fatalf("forged: %v", err)
	}
}

func run(host string, port int) error {
	authCfg := auth.FromEnv()
	if !authCfg.Required() {
		// opencode warns when the server is unsecured (cli/cmd/serve.ts:15).
		log.Printf("warning: OPENCODE_SERVER_PASSWORD is not set; server is unauthenticated")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	db, err := storage.Open(storage.DefaultPath())
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = db.Close() }()

	handler, err := server.New(server.Options{
		Version:  version,
		Auth:     authCfg,
		Cwd:      cwd,
		Sessions: session.NewStore(db),
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      0, // SSE streams are long-lived (plan 01 §1)
		IdleTimeout:       120 * time.Second,
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	// opencode prints this exact prefix; clients scrape it for the bound port.
	log.Printf("opencode server listening on http://%s", ln.Addr().String())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "forged: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

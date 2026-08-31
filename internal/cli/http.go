package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"master-agent/internal/api"
	"master-agent/internal/store"
)

const (
	defaultHTTPAddr    = "127.0.0.1:8080"
	defaultSecretsDir  = "/secrets"
	envHTTPAddr        = "HTTP_ADDR"
	httpShutdownWait   = 10 * time.Second
)

func resolveHTTPAddr(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v := os.Getenv(envHTTPAddr); v != "" {
		return v
	}
	return defaultHTTPAddr
}

func newAPIServer(s *store.Store, secretsDir string) *api.Server {
	return api.New(api.Config{
		Store:      s,
		SecretsDir: secretsDir,
		Token:      api.TokenFromEnv(),
		Logger:     slog.Default(),
	})
}

// startHTTPServer listens on addr until ctx is cancelled, then shuts down gracefully.
func startHTTPServer(ctx context.Context, addr string, handler http.Handler, out io.Writer) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownWait)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		select {
		case err := <-errCh:
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		case <-shutdownCtx.Done():
			return ctx.Err()
		}
	}
}

func printHTTPListening(out io.Writer, addr string) {
	fmt.Fprintf(out, "http listening on %s\n", addr)
}

package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"master-agent/internal/store"
)

func newServeCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	var addrFlag string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API and Web UI against the SQLite store",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := resolveHTTPAddr(addrFlag)

			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			apiSrv := newAPIServer(s)
			printHTTPListening(opts.Stdout, addr)
			err = startHTTPServer(ctx, addr, newHTTPHandler(apiSrv), opts.Stdout)
			if err != nil && ctx.Err() != nil {
				fmtStopped(opts.Stdout, "serve")
				return nil
			}
			return err
		},
	}

	cmd.Flags().StringVar(&addrFlag, "addr", "",
		"HTTP listen address (default 127.0.0.1:8080); overrides HTTP_ADDR env")
	return cmd
}

func fmtStopped(out io.Writer, name string) {
	fmt.Fprintf(out, "%s stopped\n", name)
}

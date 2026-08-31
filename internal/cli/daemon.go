package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"master-agent/internal/lock"
	"master-agent/internal/runner"
	"master-agent/internal/scheduler"
	"master-agent/internal/store"
)

const (
	defaultTickInterval = 30 * time.Second
	envTickInterval     = "TICK_INTERVAL"
)

func newDaemonCmd(opts Options, openStore func() (*store.Store, error)) *cobra.Command {
	var tickFlag string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the scheduler daemon (tick loop + SSH runs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			tick, err := resolveTickInterval(tickFlag)
			if err != nil {
				return err
			}

			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			d := &scheduler.Daemon{
				Store:  s,
				Locks:  lock.NewManager(s, nil),
				Runner: &runner.SSHRunner{},
				Config: scheduler.Config{TickInterval: tick},
			}

			dbPath, _ := cmd.Flags().GetString("db")
			fmt.Fprintf(opts.Stdout, "daemon starting (tick=%s db=%s)\n", tick, dbPath)
			err = d.Run(ctx)
			if err != nil && ctx.Err() != nil {
				fmt.Fprintf(opts.Stdout, "daemon stopped\n")
				return nil
			}
			return err
		},
	}

	cmd.Flags().StringVar(&tickFlag, "tick-interval", "",
		"scheduler poll interval (duration, e.g. 30s); overrides TICK_INTERVAL env; default 30s")
	return cmd
}

func resolveTickInterval(flagValue string) (time.Duration, error) {
	if flagValue != "" {
		d, err := time.ParseDuration(flagValue)
		if err != nil {
			return 0, fmt.Errorf("invalid --tick-interval: %w", err)
		}
		if d <= 0 {
			return 0, fmt.Errorf("--tick-interval must be positive")
		}
		return d, nil
	}
	if v := os.Getenv(envTickInterval); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("invalid %s: %w", envTickInterval, err)
		}
		if d <= 0 {
			return 0, fmt.Errorf("%s must be positive", envTickInterval)
		}
		return d, nil
	}
	return defaultTickInterval, nil
}

// Package cli implements the master-agent command-line interface.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"master-agent/internal/store"
)

const defaultDBPath = "/data/master-agent.db"

// Options configures the CLI for tests (custom IO and store opener).
type Options struct {
	Stdout io.Writer
	Stderr io.Writer
	Open   func(path string) (*store.Store, error)
}

func (o Options) withDefaults() Options {
	if o.Stdout == nil {
		o.Stdout = os.Stdout
	}
	if o.Stderr == nil {
		o.Stderr = os.Stderr
	}
	if o.Open == nil {
		o.Open = store.Open
	}
	return o
}

// Run is the CLI entrypoint used by cmd/master-agent.
func Run(args []string) error {
	return RunWithOptions(args, Options{})
}

// RunWithOptions runs the CLI with injectable dependencies (for tests).
func RunWithOptions(args []string, opts Options) error {
	opts = opts.withDefaults()
	root := newRootCmd(opts)
	root.SetArgs(args)
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)
	return root.Execute()
}

type rootFlags struct {
	dbPath string
}

func newRootCmd(opts Options) *cobra.Command {
	flags := &rootFlags{}
	root := &cobra.Command{
		Use:           "master-agent",
		Short:         "Schedule SSH runs of remote CLI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&flags.dbPath, "db", defaultDBPath, "path to SQLite database")

	openStore := func() (*store.Store, error) {
		s, err := opts.Open(flags.dbPath)
		if err != nil {
			return nil, fmt.Errorf("open database %s: %w", flags.dbPath, err)
		}
		return s, nil
	}

	root.AddCommand(newProjectCmd(opts, openStore))
	root.AddCommand(newTaskCmd(opts, openStore))
	root.AddCommand(newRunCmd(opts, openStore))
	root.AddCommand(newDaemonCmd(opts, openStore))
	return root
}

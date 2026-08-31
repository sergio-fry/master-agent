// Package cli will host the master-agent command-line interface (cobra/urfave).
package cli

import "fmt"

// Run is the CLI entrypoint. Full command wiring lands in a later task.
func Run(args []string) error {
	if len(args) == 0 {
		fmt.Println("master-agent")
		return nil
	}
	return fmt.Errorf("unknown command %q (CLI not implemented yet)", args[0])
}

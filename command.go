package gaz

import (
	"github.com/petabytecl/gaz/di"
	"github.com/spf13/cobra"
)

// CommandArgs holds the Cobra command and arguments for the current execution.
// It is available in the DI container for services that need access to CLI inputs.
type CommandArgs struct {
	Command *cobra.Command
	Args    []string
}

// GetArgs retrieves the CLI arguments from the DI container.
// It returns nil if CommandArgs is not available in the container.
func GetArgs(c *di.Container) []string {
	if c == nil {
		return nil
	}

	args, err := Resolve[*CommandArgs](c)
	if err != nil {
		return nil
	}
	if args == nil {
		return nil
	}

	return args.Args
}

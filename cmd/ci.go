package cmd

import (
	"github.com/spf13/cobra"
)

func getCISubCommands() *cobra.Command {
	command := &cobra.Command{
		Use:   "ci",
		Short: "Groups the ci sub-commands",
	}

	command.AddCommand(getCIAutoReleasePlanCommand())
	command.AddCommand(getCIDetectChangesCommand())
	command.AddCommand(getCIReleaseCommand())

	return command
}

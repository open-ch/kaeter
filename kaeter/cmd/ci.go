package cmd

import (
	"github.com/spf13/cobra"
)

func getCISubCommands() *cobra.Command {
	ciCmd := &cobra.Command{
		Use:   "ci",
		Short: "Groups the ci sub-commands",
	}

	ciCmd.AddCommand(getCIAutoReleasePlanCommand())
	ciCmd.AddCommand(getCIReleaseCommand())

	return ciCmd
}

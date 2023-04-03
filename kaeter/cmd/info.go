package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/kaeter/modules"
)

func getInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "Gather and print info about a kaeter module",
		PreRunE: validateAllPathFlags,
		Run: func(_ *cobra.Command, args []string) {
			modulePaths := viper.GetStringSlice("path")
			for _, modulePath := range modulePaths {
				modules.PrintModuleInfo(modulePath)
			}
		},
	}

	return cmd
}

package cmd

import (
	"os"

	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/spf13/cobra"
)

func init() {
	// For a SemVer versioned module, should the minor or major be bumped?
	var moduleID string

	// TODO check repo for existing modules
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise a module's versions.yml file.",
		Long:  `Initialise a module's versions.yml file.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runInit(moduleID)
			if err != nil {
				logger.Errorf("Init failed: %s", err)
				os.Exit(1)
			}
		},
	}

	initCmd.Flags().StringVar(&moduleID, "id", "",
		"The identification string for this module. Something looking like maven coordinates is preferred.")

	initCmd.MarkFlagRequired("id")

	rootCmd.AddCommand(initCmd)
}

func runInit(moduleID string) error {
	logger.Infof("Initialising versions.yml file at: %s", modulePath)
	_, err := kaeter.Initialise(modulePath, moduleID)
	return err
}

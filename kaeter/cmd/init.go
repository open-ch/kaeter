package cmd

import (
	"errors"
	"os"

	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/spf13/cobra"
)

func init() {
	// Identifier for the module: can be maven style groupId:moduleId or any string without a colon.
	var moduleID string

	// What versioning scheme to use
	var versioningScheme string

	// TODO check repo for existing modules
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise a module's versions.yaml file.",
		Long:  `Initialise a module's versions.yaml file.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runInit(moduleID, versioningScheme)
			if err != nil {
				logger.Errorf("Init failed: %s", err)
				os.Exit(1)
			}
		},
	}

	initCmd.Flags().StringVar(&moduleID, "id", "",
		"The identification string for this module. Something looking like maven coordinates is preferred.")

	initCmd.Flags().StringVar(&versioningScheme, "scheme", "SemVer",
		"Versioning scheme to use: one of SemVer, CalVer or AnyStringVer. Defaults to SemVer.")

	initCmd.MarkFlagRequired("id")

	rootCmd.AddCommand(initCmd)
}

func runInit(moduleID string, versioningScheme string) error {
	if len(modulePaths) != 1 {
		return errors.New("init command only supports exactly one path")
	}

	modulePath := modulePaths[0]
	logger.Infof("Initialising versions.yaml file at: %s", modulePath)
	_, err := kaeter.Initialise(modulePath, moduleID, versioningScheme)
	return err
}

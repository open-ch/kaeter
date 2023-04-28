package cmd

import (
	"errors"

	"github.com/open-ch/kaeter/kaeter/modules"

	"github.com/spf13/cobra"
)

func getInitCommand() *cobra.Command {
	// Identifier for the module: can be maven style groupId:moduleId or any string without a colon.
	var moduleID string

	// What versioning scheme to use
	var versioningScheme string

	// If we should init or touch the readme and/or changelog upon init
	var noReadme bool
	var noChangelog bool

	// TODO check repo for existing modules
	initCmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialise a module's versions.yaml file.",
		Long:    "Initialise a module's versions.yaml file.",
		PreRunE: validateAllPathFlags,
		Run: func(cmd *cobra.Command, args []string) {
			err := runInit(moduleID, versioningScheme, noReadme, noChangelog)
			if err != nil {
				logger.Fatalf("Init failed: %s\n", err)
			}
		},
	}

	initCmd.Flags().StringVar(&moduleID, "id", "",
		"The identification string for this module. Something looking like maven coordinates is preferred.")

	initCmd.Flags().StringVar(&versioningScheme, "scheme", "SemVer",
		"Versioning scheme to use: one of SemVer, CalVer or AnyStringVer. Defaults to SemVer.")

	initCmd.Flags().BoolVar(&noReadme, "no-readme", false, "Should an empty README.md file be created next to the module configuration if none exists."+
		"If it exists and a Changelog is being created, a link to the changelog will be appended to the readme.")

	initCmd.Flags().BoolVar(&noChangelog, "no-changelog", false, "Should an empty CHANGELOG.md file be created next to the module configuration if none exists."+
		"If it is created and a README file exists, a link to the changelog file will be appended to the readme.")

	_ = initCmd.MarkFlagRequired("id")

	return initCmd
}

func runInit(moduleID, versioningScheme string, noReadme, noChangelog bool) error {
	if len(modulePaths) != 1 {
		return errors.New("init command only supports exactly one path")
	}

	modulePath := modulePaths[0]
	logger.Infof("Initialising versions.yaml file at: %s", modulePath)
	_, err := modules.Initialise(modulePath, moduleID, versioningScheme, !noReadme, !noChangelog)
	return err
}

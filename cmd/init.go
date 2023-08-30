package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/open-ch/kaeter//log"
	"github.com/open-ch/kaeter//modules"
)

func getInitCommand() *cobra.Command {
	var moduleID string
	var versioningScheme string
	var noReadme bool
	var noChangelog bool

	initCmd := &cobra.Command{
		Use:     "init",
		Short:   "Initialise a module's versions.yaml file.",
		PreRunE: validateAllPathFlags,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(moduleID, versioningScheme, noReadme, noChangelog)
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
	log.Infof("Initialising versions.yaml file at: %s", modulePath)
	_, err := modules.Initialise(modulePath, moduleID, versioningScheme, !noReadme, !noChangelog)
	return err
}

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

func getInitCommand() *cobra.Command {
	var moduleID string
	var versioningScheme string
	var noReadme bool
	var noChangelog bool

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a module's versions.yaml file.",
		// TODO can we generate custom help text that dynamically generates a list of the available templates based on config?
		Long: `Initialize a new kaeter module using the given path and id.
A kaeter module has 4 key components:
- versions.yaml
- README.md
- A Makefile (Makefile.kaeter or Makefile) with the default targets
- A changelog (different formats supported)

init must create the versions.yaml file, and will fail in case of an existing file.

Basic README.md and a CHANGELOG.md will be created if none are found, unless using flags to skip
their creation. When both are created the CHANGELOG.md will be linked from the README.md to avoid
deadlinks.
`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, _ []string) error {
			modulePaths := viper.GetStringSlice("path")
			if len(modulePaths) != 1 {
				return errors.New("init command only supports exactly one path value")
			}

			moduleConfig := modules.InitializationConfig{
				InitChangelog:    !noChangelog,
				InitReadme:       !noReadme,
				ModuleID:         moduleID,
				ModulePath:       modulePaths[0],
				VersioningScheme: versioningScheme,
			}

			log.Info("Initializing new kaeter module", "moduleID", moduleConfig.ModuleID, "modulePath", moduleConfig.ModulePath)
			_, err := modules.Initialize(moduleConfig)
			return err
		},
	}

	flags := initCmd.Flags()

	flags.StringVar(&moduleID, "id", "",
		"The identification string for this module. Something looking like maven coordinates is preferred.")
	err := initCmd.MarkFlagRequired("id")
	if err != nil {
		log.Warn("Error with required id flag", "err", err)
	}
	flags.StringVar(&versioningScheme, "scheme", "SemVer",
		"Versioning scheme to use: one of SemVer, CalVer or AnyStringVer. Defaults to SemVer.")
	flags.BoolVar(&noReadme, "no-readme", false, "Skip README.md creation even if none exists.")
	flags.BoolVar(&noChangelog, "no-changelog", false, "Skip CHANGELOG.md creation even if none exists. ")
	// TODO add --no-makefile flag
	// TODO add --template=type flag

	return initCmd
}

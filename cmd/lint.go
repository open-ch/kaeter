package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/lint"
)

func getLintCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "lint --path path/to/check/from",
		Short: "Basic quality checks for the detected modules.",
		Long: `Finds the repository root, then detects Kaeter modules,
for every kaeter-managed module (which has a versions.yaml file) the following
are checked:
- the existence of README.md
- the existence of a changelog (defaults to CHANGELOG.md)
- the changelog is up-to-date with versions.yml (for releases)

Note that it will stop at the first error and not check remaining existing modules

Previously called "kaeter-police check".`,
		Run: func(_ *cobra.Command, _ []string) {
			repositoryRoot := viper.GetString("repoRoot")
			err := lint.CheckModulesStartingFrom(repositoryRoot)
			if err != nil {
				// revive:disable-next-line
				log.Fatalf("Check failed: %s", err)
			}
			log.Info("No issues detected.")
		},
	}

	return command
}

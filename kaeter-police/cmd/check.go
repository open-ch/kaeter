package cmd

import (
	"log"

	kaeterPolice "github.com/open-ch/kaeter/kaeter-police/pkg/kaeterpolice"

	"github.com/spf13/cobra"
)

func getCheckCommand() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check --path path/to/check/from",
		Short: "Basic quality checks for the detected modules.",
		Long: `Check modules detected from the starting path meet base quality requirements.
For every kaeter-managed package (which has a versions.yaml file) the following
are checked:
- the existence of README.md
- the existence of a changelog (defaults to CHANGELOG.md)
- the changelog is up-to-date with versions.yml (for releases)

Note that it will stop on the first issue and not list all existing issues`,
		Run: func(cmd *cobra.Command, args []string) {
			err := kaeterPolice.CheckModulesStartingFrom(rootPath)
			if err != nil {
				log.Fatalf("Check failed: %s", err)
			}
		},
	}

	return checkCmd
}

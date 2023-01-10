package cmd

import (
	"log"

	kaeterPolice "github.com/open-ch/kaeter/kaeter/pkg/lint"

	"github.com/spf13/cobra"
)

func getCheckCommand() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check --path path/to/check/from",
		Short: "Basic quality checks for the detected modules.",
		Long: `Finds the repository root, then detects Kaeter modules,
for every kaeter-managed module (which has a versions.yaml file) the following
are checked:
- the existence of README.md
- the existence of a changelog (defaults to CHANGELOG.md)
- the changelog is up-to-date with versions.yml (for releases)

Note that it will stop at the first error and not check remaining existing modules`,
		Run: func(cmd *cobra.Command, args []string) {
			err := kaeterPolice.CheckModulesStartingFrom(rootPath)
			if err != nil {
				// revive:disable-next-line
				log.Fatalf("Check failed: %s", err)
			}
		},
	}

	return checkCmd
}

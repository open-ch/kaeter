package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-ch/kaeter/ci"
)

func getCIReleaseCommand() *cobra.Command {
	var dryrun bool

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Performs a ci release of a single module",
		RunE: func(_ *cobra.Command, _ []string) error {
			if len(modulePaths) != 1 {
				return fmt.Errorf("only a single module can be released at a time, got: %d", len(modulePaths))
			}

			rc := &ci.ReleaseConfig{
				DryRun:     dryrun,
				ModulePath: modulePaths[0],
			}
			return rc.ReleaseSingleModule()
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&dryrun, "dry-run", false, "Build and test but don't push the release")
	// add --version flag or --snapshot flag to use this command for snapshots as well.

	return cmd
}

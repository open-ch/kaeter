package cmd

import (
	"github.com/spf13/cobra"

	"github.com/open-ch/kaeter/kaeter/ci"
)

func getCIReleaseCommand() *cobra.Command {
	var dryrun bool

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Performs a ci release of a single module",
		Run: func(cmd *cobra.Command, args []string) {
			if len(modulePaths) != 1 {
				logger.Fatalf("Only a single module can be released at a time, got: %d\n", len(modulePaths))
			}

			rc := &ci.ReleaseConfig{
				DryRun:     dryrun,
				ModulePath: modulePaths[0],
				Logger:     logger,
			}
			err := rc.ReleaseSingleModule()
			if err != nil {
				logger.Fatalf("module release failed: %s\n", err)
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&dryrun, "dry-run", false, "Build and test but don't push the release")
	// add --version flag or --snapshot flag to use this command for snapshots as well.

	return cmd
}

package cmd

import (
	"os"

	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/spf13/cobra"
)

func getReleaseCommand() *cobra.Command {
	var really bool
	var nocheckout bool
	var skipModules []string

	releaseCmd := &cobra.Command{
		Use:   "release",
		Short: "Executes a release plan.",
		Long: `Executes a release plan: currently such a plan can only be provided via the last commit in the repository
on which kaeter is being run. See kaeter's doc for more details.'`,
		Run: func(cmd *cobra.Command, args []string) {
			if !really {
				logger.Warnf("'really' flag is set to false: will run build and tests but no release.")
			}
			if !nocheckout {
				logger.Warnf("'nocheckout' flag is set to false: will checkout the commit hash corresponding to the version of the module.")
			}

			releaseConfig := &kaeter.ReleaseConfig{
				RepositoryRoot:  repoRoot,
				RepositoryTrunk: gitMainBranch,
				DryRun:          !really,
				SkipCheckout:    nocheckout,
				SkipModules:     skipModules,
				Logger:          logger,
			}
			err := kaeter.RunReleases(releaseConfig)
			if err != nil {
				logger.Errorf("release failed: %s", err)
				os.Exit(1)
			}
		},
	}

	releaseCmd.Flags().BoolVar(&really, "really", false,
		`If set, and if the module is using SemVer, causes a bump in the minor version of the released module.
By default the build number is incremented.`)
	releaseCmd.Flags().BoolVar(&nocheckout, "nocheckout", false,
		`If set, no checkout of the commit hash corresopnding to the version of the module will be made before
releasing.`)
	releaseCmd.Flags().StringArrayVar(&skipModules, "skip-module", []string{},
		`List of kaeter module IDs to skip even if present in release plan`)

	return releaseCmd
}

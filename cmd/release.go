package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/log"
)

func getReleaseCommand() *cobra.Command {
	var really bool
	var nocheckout bool
	var skipModules []string
	var commitMessage string

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Executes a release plan.",
		Long: `Executes a release plan: currently such a plan can only be provided via the last commit in the repository
on which kaeter is being run. See kaeter's doc for more details.'`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !really {
				log.Warn("'really' flag is set to false: will run build and tests but no release.")
			}
			if !nocheckout {
				log.Warn("'nocheckout' flag is set to false: will checkout the commit hash corresponding to the version of the module.")
			}

			releaseConfig := &actions.ReleaseConfig{
				RepositoryRoot:       viper.GetString("repoRoot"),
				RepositoryTrunk:      viper.GetString("git.main.branch"),
				DryRun:               !really,
				SkipCheckout:         nocheckout,
				SkipModules:          skipModules,
				ReleaseCommitMessage: commitMessage,
			}
			err := actions.RunReleases(releaseConfig)
			if err != nil {
				return fmt.Errorf("release failed: %w", err)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&really, "really", false,
		`If set, and if the module is using SemVer, causes a bump in the minor version of the released module.
By default the build number is incremented.`)
	flags.BoolVar(&nocheckout, "nocheckout", false,
		`If set, no checkout of the commit hash corresopnding to the version of the module will be made before
releasing.`)
	flags.StringArrayVar(&skipModules, "skip-module", []string{}, "List of kaeter module IDs to skip even if present in release plan")
	flags.StringVar(&commitMessage, "commit-message", "", "Read release plan from this string instead of git")

	cmd.MarkFlagsMutuallyExclusive("really", "commit-message")

	return cmd
}

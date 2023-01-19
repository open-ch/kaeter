package cmd

import (
	"github.com/spf13/cobra"

	actions "github.com/open-ch/kaeter/kaeter/pkg/actions"
)

func getPrepareCommand() *cobra.Command {
	var major bool
	var minor bool
	var releaseFrom string
	var skipLint bool
	var userProvidedVersion string

	prepareCmd := &cobra.Command{
		Use:   "prepare -p path/to/module [--minor|--major|--version=1.4.2]",
		Short: "Prepare the release of the specified module.",
		Long: `Prepare the release of the specified module based on the module's versions.yaml file
and the flags passed to it, this command will:
 - determine the next version to be released, using either SemVer of CalVer
 - update the versions.yaml file for the relevant project
 - serialize the release plan to a commit`,
		PreRunE: validateAllPathFlags,
		Run: func(cmd *cobra.Command, args []string) {
			prepareConfig := &actions.PrepareReleaseConfig{
				BumpMajor:           major,
				BumpMinor:           minor,
				ModulePaths:         modulePaths,
				RepositoryRef:       gitMainBranch,
				RepositoryRoot:      repoRoot,
				UserProvidedVersion: userProvidedVersion,
				Logger:              logger,
				SkipLint:            skipLint,
			}

			if releaseFrom != "" {
				prepareConfig.RepositoryRef = releaseFrom
			}

			err := actions.PrepareRelease(prepareConfig)
			if err != nil {
				logger.Fatal(err)
			}
		},
	}

	prepareCmd.Flags().BoolVar(&minor, "minor", false,
		"If set, and if the module is using SemVer, causes a bump in the minor version of the released module.")

	prepareCmd.Flags().BoolVar(&major, "major", false,
		"If set, and if the module is using SemVer, causes a bump in the major version of the released module.")

	prepareCmd.Flags().StringVar(&userProvidedVersion, "version", "",
		"If specified, this version will be used for the prepared release, instead of deriving one.")

	prepareCmd.Flags().StringVar(&releaseFrom, "releaseFrom", "",
		`Git ref to resolve the commit hash to release from.
Default: git-main-branch from the config (can be a branch, a tag or a commit hash).`)

	prepareCmd.Flags().BoolVar(&skipLint, "skip-lint", false,
		"Skips validation of the release, use at your own risk for broken builds.")

	return prepareCmd
}

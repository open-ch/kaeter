package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/modules"
)

func getPrepareCommand() *cobra.Command {
	var major bool
	var minor bool
	var releaseFrom string
	var skipLint bool
	var userProvidedVersion string

	cmd := &cobra.Command{
		Use:   "prepare -p path/to/module [--minor|--major|--version=1.4.2]",
		Short: "Prepare the release of the specified module.",
		Long: `Prepare the release of the specified module based on the module's versions.yaml file
and the flags passed to it, this command will:
 - determine the next version to be released, using either SemVer of CalVer
 - update the versions.yaml file for the relevant project
 - serialize the release plan to a commit`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, args []string) error {
			var bumpType modules.SemVerBump
			if major {
				bumpType = modules.BumpMajor
			} else if minor {
				bumpType = modules.BumpMinor
			}

			prepareConfig := &actions.PrepareReleaseConfig{
				BumpType:            bumpType,
				ModulePaths:         modulePaths,
				RepositoryRef:       viper.GetString("git.main.branch"),
				RepositoryRoot:      repoRoot,
				UserProvidedVersion: userProvidedVersion,
				SkipLint:            skipLint,
			}

			if releaseFrom != "" {
				prepareConfig.RepositoryRef = releaseFrom
			}

			return actions.PrepareRelease(prepareConfig)
		},
	}

	flags := cmd.Flags()

	flags.BoolVar(&minor, "minor", false,
		"If set, and if the module is using SemVer, causes a bump in the minor version of the released module.")
	flags.BoolVar(&major, "major", false,
		"If set, and if the module is using SemVer, causes a bump in the major version of the released module.")
	flags.StringVar(&userProvidedVersion, "version", "",
		"If specified, this version will be used for the prepared release, instead of deriving one.")
	flags.StringVar(&releaseFrom, "releaseFrom", "",
		`Git ref to resolve the commit hash to release from.
Default: git-main-branch from the config (can be a branch, a tag or a commit hash).`)
	flags.BoolVar(&skipLint, "skip-lint", false,
		"Skips validation of the release, use at your own risk for broken builds.")

	cmd.MarkFlagsMutuallyExclusive("minor", "major", "version")

	return cmd
}

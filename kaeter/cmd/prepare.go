package cmd

import (
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/spf13/cobra"
)

func getPrepareCommand() *cobra.Command {
	// For a SemVer versioned module, should the minor or major be bumped?
	var minor bool
	var major bool

	// Version passed via CLI
	var userProvidedVersion string

	// Branch, Tag or Commit to do a release from:
	var releaseFrom string

	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the release of the specified module.",
		Long: `Prepare the release of the specified module:

Based on the module's versions.yaml file and the flags passed to it, this command will:'
 - determine the next version to be released, using either SemVer of CalVer;
 - update the versions.yaml file for the relevant project
 - serialize the release plan to a commit`,
		PreRunE: validateAllPathFlags,
		Run: func(cmd *cobra.Command, args []string) {
			prepareConfig := &kaeter.PrepareReleaseConfig{
				BumpMajor:           major,
				BumpMinor:           minor,
				ModulePaths:         modulePaths,
				RepositoryRef:       gitMainBranch,
				RepositoryRoot:      repoRoot,
				UserProvidedVersion: userProvidedVersion,
				Logger:              logger,
			}

			if releaseFrom != "" {
				prepareConfig.RepositoryRef = releaseFrom
			}

			err := kaeter.PrepareRelease(prepareConfig)
			if err != nil {
				logger.Fatalf("Prepare failed: %s\n", err)
			}
		},
	}

	prepareCmd.Flags().BoolVar(&minor, "minor", false,
		`If set, and if the module is using SemVer, causes a bump in the minor version of the released module.
By default the build number is incremented.`)

	prepareCmd.Flags().BoolVar(&major, "major", false,
		`If set, and if the module is using SemVer, causes a bump in the major version of the released module.
By default the build number is incremented.`)

	prepareCmd.Flags().StringVar(&userProvidedVersion, "version", "",
		"If specified, this version will be used for the prepared release, instead of deriving one.")

	prepareCmd.Flags().StringVar(&releaseFrom, "releaseFrom", "",
		`If specified, use this identifier to resolve the commit id from which to do the release.
Can be a branch, a tag or a commit id.
Note that it is wise to release a commit that already exists in a remote.
Defaults to the value of the global --git-main-branch option.`)

	return prepareCmd
}

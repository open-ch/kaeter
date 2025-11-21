package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/log"
)

func getAutoreleaseCommand() *cobra.Command {
	skipLint := false
	autoreleaseCmd := &cobra.Command{
		Use:     "autorelease --path <PATH> --version <VERSION>",
		Aliases: []string{"ar"},
		Short:   "Defines a release for the current branch/code review",
		Long: `Configures the given module such that when the branch
is merged back to trunk the module will be released. This is the
local counterpart of kaeter ci autorelease plan to be used in a pipeline
to release on merge.

- Can be called multiple times for multiple modules to be released
`,
		PreRunE: validateAllPathFlags,
		RunE: func(cmd *cobra.Command, _ []string) error {
			log.Debug("Viper settings", "allSettings", viper.AllSettings())

			if len(viper.GetStringSlice("path")) != 1 {
				log.Debug("Available paths", "paths", viper.GetStringSlice("path"))
				return errors.New("invalid number of paths, only 1 path supported for autorelease")
			}

			version, err := cmd.Flags().GetString("version")
			if err != nil {
				return fmt.Errorf("autorelease failed: unable to parse version: %w", err)
			}

			tags, err := cmd.Flags().GetStringSlice("tags")
			if err != nil {
				return fmt.Errorf("autorelease failed: unable to parse tags: %w", err)
			}

			// Check if tags flag was explicitly provided
			tagsChanged := cmd.Flags().Changed("tags")
			var tagsPtr *[]string
			if tagsChanged {
				tagsPtr = &tags
			}

			modulePath, err := resolveModuleAbsPath()
			if err != nil {
				return fmt.Errorf("autorelease failed: %w", err)
			}

			config := &actions.AutoReleaseConfig{
				ModulePath:     modulePath,
				RepositoryRef:  viper.GetString("git.main.branch"),
				RepositoryRoot: viper.GetString("reporoot"),
				ReleaseVersion: version,
				Tags:           tagsPtr,
				SkipLint:       skipLint,
			}

			return actions.AutoRelease(config)
		},
	}

	autoreleaseCmd.Flags().StringP("version", "v", "",
		"Version number to use when the release will be triggered on CI.")
	autoreleaseCmd.Flags().StringSlice("tags", nil,
		"Comma-separated list of custom tags for this release (e.g., production,stable,lts).")
	autoreleaseCmd.Flags().BoolVar(&skipLint, "skip-lint", false,
		"Skips validation of the release, use at your own risk for broken builds.")

	return autoreleaseCmd
}

func resolveModuleAbsPath() (string, error) {
	rawModulePath := viper.GetStringSlice("path")[0]

	absModulePath, err := filepath.Abs(rawModulePath)
	if err != nil {
		return "", fmt.Errorf("unable to resolve absolute path from %s: %w", rawModulePath, err)
	}
	return absModulePath, nil
}

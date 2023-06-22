package cmd

import (
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
		Use:   "autorelease --path [PATH] --version [VERSION]",
		Aliases: []string{"ar"},
		Short: "Defines a release for the current branch/code review",
		Long: `Configures the given module such that when the branch
is merged back to trunk the module will be released. This is the
local counterpart of kaeter ci autorelease plan to be used in a pipeline
to release on merge.

- Can be called multiple times for multiple modules to be released
`,
		PreRunE: validateAllPathFlags,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debugf("Viper settings: %v", viper.AllSettings())

			if len(viper.GetStringSlice("path")) != 1 {
				log.Debugf("Available paths %v", viper.GetStringSlice("path"))
				return fmt.Errorf("Invalid number of paths, only 1 path supported for autorelease")
			}

			version, err := cmd.Flags().GetString("version")
			if err != nil {
				return fmt.Errorf("Autorelease failed: unable to parse version: %w", err)
			}

			modulePath, err := resolveModuleAbsPath()
			if err != nil {
				return fmt.Errorf("Autorelease failed: %w", err)
			}

			config := &actions.AutoReleaseConfig{
				ModulePath:     modulePath,
				RepositoryRef:  viper.GetString("git.main.branch"),
				RepositoryRoot: viper.GetString("reporoot"),
				ReleaseVersion: version,
				SkipLint:       skipLint,
			}

			return actions.AutoRelease(config)
			if err != nil {
				return fmt.Errorf("Autorelease failed: %w", err)
			}
			return nil
		},
	}

	autoreleaseCmd.Flags().StringP("version", "v", "",
		"Version number to use when the release will be triggered on CI.")
	autoreleaseCmd.Flags().BoolVar(&skipLint, "skip-lint", false,
		"Skips validation of the release, use at your own risk for broken builds.")

	return autoreleaseCmd
}

func resolveModuleAbsPath() (string, error) {
	rawModulePath := viper.GetStringSlice("path")[0]

	absModulePath, err := filepath.Abs(rawModulePath)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve absolute path from %s: %w", rawModulePath, err)
	}
	return absModulePath, nil
}

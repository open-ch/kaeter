package cmd

import (
	"github.com/spf13/viper"

	"github.com/spf13/cobra"

	actions "github.com/open-ch/kaeter/kaeter/actions"
)

func getAutoreleaseCommand() *cobra.Command {
	autoreleaseCmd := &cobra.Command{
		Use:   "autorelease --path [PATH] --version [VERSION]",
		Short: "Defines a release for the current branch/code review",
		Long: `Configures the given module such that when the branch
is merged back to trunk the module will be released. This is the
local counterpart of kaeter ci autorelease plan to be used in a pipeline
to release on merge.

- Can be called multiple times for multiple modules to be released
`,
		PreRunE: validateAllPathFlags,
		Run: func(cmd *cobra.Command, args []string) {
			logger.Debugf("Viper settings: %v", viper.AllSettings())

			if len(viper.GetStringSlice("path")) != 1 {
				logger.Debugf("Available paths %v", viper.GetStringSlice("path"))
				logger.Fatalln("Invalid number of paths, only 1 path supported for autorelease")
			}

			version, err := cmd.Flags().GetString("version")
			if err != nil {
				logger.Fatalf("Autorelease unable to parse version: %s\n", err)
			}

			config := &actions.AutoReleaseConfig{
				ModulePath:     viper.GetStringSlice("path")[0],
				RepositoryRef:  viper.GetString("git.main.branch"),
				RepositoryRoot: viper.GetString("reporoot"),
				ReleaseVersion: version,
				Logger:         logger,
			}

			err = actions.AutoRelease(config)
			if err != nil {
				logger.Fatalf("Autorelease failed: %s\n", err)
			}
		},
	}

	autoreleaseCmd.Flags().StringP("version", "v", "",
		"Version number to use when the release will be triggered on CI.")
	_ = autoreleaseCmd.MarkFlagRequired("version")

	return autoreleaseCmd
}

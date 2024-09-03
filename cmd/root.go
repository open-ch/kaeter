package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"

	charmlog "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Mapping from flags names to config file names
// to sync between viper and cobra
var configMap = map[string]string{ //nolint:gochecknoglobals
	"git-main-branch": "git.main.branch",
}

// Execute runs the whole enchilada, baby!
func Execute() error {
	charmlog.SetReportTimestamp(false)

	rootCmd := &cobra.Command{
		Use:   "kaeter",
		Short: "kaeter handles the releasing and versioning of your modules within a fat repo.",
		Long: `kaeter offers a standard approach for releasing and versioning arbitrary artifacts.
Its goal is to provide a 'descriptive release' process, in which developers request the release of given artifacts,
and upon acceptance of the request, a separate build infrastructure is in charge of carrying out the build.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initializeConfig(cmd)
		},
	}

	// The default completions don't work very well, hide them.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	topLevelFlags := rootCmd.PersistentFlags()
	topLevelFlags.StringArrayP("path", "p", []string{"."},
		`Path to where kaeter must work from. This is either the module for which a release is required,
or the repository for which a release plan must be executed.
Multiple paths can be passed for subcommands that support it.`)
	err := viper.BindPFlag("path", topLevelFlags.Lookup("path"))
	if err != nil {
		log.Error("Unable to parse path flag", "err", err)
		return err // path is an important flag, don't continue if it can't be parsed.
	}
	topLevelFlags.BoolP("debug", "d", false, `Sets logs to be more verbose`)
	err = viper.BindPFlag("debug", topLevelFlags.Lookup("debug"))
	if err != nil {
		log.Error("Unable to parse debug flag", "err", err)
	}
	topLevelFlags.String("log-level", "", `Sets a specific logger output level`)
	err = viper.BindPFlag("log-level", topLevelFlags.Lookup("log-level"))
	if err != nil {
		log.Error("Unable to parse debug flag", "err", err)
	}
	topLevelFlags.String("git-main-branch", "",
		`Defines the main branch of the repository, can also be set in the configuration file as "git.main.branch".`)

	rootCmd.AddCommand(getAutoreleaseCommand())
	rootCmd.AddCommand(getCISubCommands())
	rootCmd.AddCommand(getInfoCommand())
	rootCmd.AddCommand(getInitCommand())
	rootCmd.AddCommand(getLintCommand())
	rootCmd.AddCommand(getPrepareCommand())
	rootCmd.AddCommand(getReadPlanCommand())
	rootCmd.AddCommand(getReleaseCommand())

	return rootCmd.Execute()
}

func initializeConfig(cmd *cobra.Command) error {
	modulePaths := viper.GetStringSlice("path")
	repoRoot := getRepoRoot(modulePaths)
	if repoRoot == "" {
		log.Warn("Unable to determine repo root based on current working directory or path(s)")
	}

	configPath := path.Join(repoRoot, ".kaeter.config.yaml")

	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok { //nolint:errorlint // ConfigFileNotFoundError isn't of type error
			log.Warnf("Failed to parse config at %s: %v", configPath, err)
		}
	}

	viper.Set("repoRoot", repoRoot)

	// This makes git.main.branch in the yaml config available to
	// rootCmd.PersistentFlags() transparently
	syncViperToCommandFlags(cmd)

	log.Initialize()

	return nil
}

func getRepoRoot(paths []string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Warn("Unable to resolve current working directory, skipping root repo resolution from cwd")
	} else {
		wdRepo, err := git.ShowTopLevel(cwd)
		if err == nil {
			return wdRepo
		}
		log.Warn("Unable to resolve repository root from working directory, fallback to path flags")
	}

	for _, modulePath := range paths {
		moduleRepo, err := git.ShowTopLevel(modulePath)
		if err == nil {
			return moduleRepo
		}
		log.Debug("Unable to resolve repository root from --path", "path", modulePath)
	}

	return ""
}

// validateAllPathFlags is used as a PreRunE hook for cobra.Command definitions
// so `_ *cobra.Command, _ []string` are required even if we don't use them.
func validateAllPathFlags(_ *cobra.Command, _ []string) error {
	paths := viper.GetStringSlice("path")

	if len(paths) == 0 {
		return errors.New("at least one --path/-p flag is required")
	}

	for _, modulePath := range paths {
		moduleRepo, err := git.ShowTopLevel(modulePath)
		if err != nil {
			return fmt.Errorf("unable find module in repository: %s\n%w", modulePath, err)
		}

		if viper.GetString("repoRoot") != moduleRepo {
			return errors.New("all paths have to be in the same repository")
		}
	}

	return nil
}

func syncViperToCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if entry, ok := configMap[f.Name]; ok && !f.Changed && viper.IsSet(entry) {
			val := viper.GetString(entry)
			_ = cmd.Flags().Set(f.Name, val) //nolint:errcheck // frivolous errors from flags
		}
	})
}

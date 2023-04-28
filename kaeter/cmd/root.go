package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/open-ch/go-libs/gitshell"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Mapping from flags names to config file names
// to sync between viper and cobra
var configMap = map[string]string{
	"git-main-branch": "git.main.branch",
}

var (
	// Points to the modules to be released
	modulePaths   []string
	gitMainBranch string
	repoRoot      string

	logger = log.New()
)

// Execute runs the whole enchilada, baby!
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "kaeter",
		Short: "kaeter handles the releasing and versioning of your modules within a fat repo.",
		Long: `kaeter offers a standard approach for releasing and versioning arbitrary artifacts.
Its goal is to provide a 'descriptive release' process, in which developers request the release of given artifacts,
and upon acceptance of the request, a separate build infrastructure is in charge of carrying out the build.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd)
		},
	}

	// The default completions don't work very well, hide them.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	topLevelFlags := rootCmd.PersistentFlags()

	topLevelFlags.StringArrayVarP(&modulePaths, "path", "p", []string{},
		`Path to where kaeter must work from. This is either the module for which a release is required,
or the repository for which a release plan must be executed.
Multiple paths can be passed for subcommands that support it.`)
	err := viper.BindPFlag("path", topLevelFlags.Lookup("path"))
	if err != nil {
		logger.Fatalln("Unable to parse path flag", err)
	}

	topLevelFlags.BoolP("debug", "d", false, `Sets logs to be more verbose`)
	err = viper.BindPFlag("debug", topLevelFlags.Lookup("debug"))
	if err != nil {
		logger.Errorln("Unable to parse debug flag", err)
	}
	topLevelFlags.String("log-level", "", `Sets a specific logger output level`)
	err = viper.BindPFlag("log-level", topLevelFlags.Lookup("log-level"))
	if err != nil {
		logger.Errorln("Unable to parse debug flag", err)
	}

	topLevelFlags.StringVar(&gitMainBranch, "git-main-branch", "",
		`Defines the main branch of the repository, can also be set in the configuration file as "git.main.branch".`)

	rootCmd.AddCommand(getAutoreleaseCommand())
	rootCmd.AddCommand(getCISubCommands())
	rootCmd.AddCommand(getInfoCommand())
	rootCmd.AddCommand(getInitCommand())
	rootCmd.AddCommand(getLintCommand())
	rootCmd.AddCommand(getPrepareCommand())
	rootCmd.AddCommand(getReadPlanCommand())
	rootCmd.AddCommand(getReleaseCommand())

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})

	if err := rootCmd.Execute(); err != nil {
		logger.Errorln(err)
		os.Exit(-1)
	}
}

func initializeConfig(cmd *cobra.Command) error {
	repoRoot = getRepoRoot(modulePaths)
	if repoRoot == "" {
		logger.Warnf("Unable to determine repo root based on path(s)")
	}

	configPath := path.Join(repoRoot, ".kaeter.config.yaml")

	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logger.Warnf("Failed to parse config at %s: %v", configPath, err)
		}
	}

	viper.Set("repoRoot", repoRoot)

	// This makes git.main.branch in the yaml config available to
	// rootCmd.PersistentFlags() transparently
	syncViperToCommandFlags(cmd)

	if viper.GetBool("debug") {
		logger.SetLevel(log.DebugLevel)
	} else if viper.GetString("log-level") != "" {
		logLevel, err := log.ParseLevel(viper.GetString("log-level"))
		if err != nil {
			logger.Fatalln(err)
		}
		logger.Level = logLevel
	}

	return nil
}

func getRepoRoot(paths []string) string {
	for _, modulePath := range paths {
		moduleRepo, err := gitshell.GitResolveRoot(modulePath)
		if err != nil {
			continue
		}

		if repoRoot == "" {
			return moduleRepo
		}
	}

	// Note we can't use os.Getwd() as fallback because we rely on
	//     bazel run //tools/kaeter:cli
	// and that runs bazel from a sandbox rather than panta.
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
		moduleRepo, err := gitshell.GitResolveRoot(modulePath)
		if err != nil {
			return fmt.Errorf("unable to determine repository root from path: %s\n%w", modulePath, err)
		}

		if repoRoot != moduleRepo {
			return errors.New("all paths have to be in the same repository")
		}
	}

	return nil
}

func syncViperToCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if entry, ok := configMap[f.Name]; ok && !f.Changed && viper.IsSet(entry) {
			val := viper.GetString(entry)
			_ = cmd.Flags().Set(f.Name, val)
		}
	})
}

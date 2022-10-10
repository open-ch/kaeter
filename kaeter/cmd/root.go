package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/open-ch/go-libs/gitshell"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

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
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			return initializeConfig(cmd)
		},
	}

	rootCmd.PersistentFlags().StringArrayVarP(&modulePaths, "path", "p", []string{"."},
		`Path to where kaeter must work from. This is either the module for which a release is required,
or the repository for which a release plan must be executed.
Multiple paths can be passed for subcommands that support it.`)

	rootCmd.PersistentFlags().StringVar(&gitMainBranch, "git-main-branch", "origin/master",
		`Defines the main branch of the repository, can also be set in the configuration file as "git.main.branch".`)

	rootCmd.AddCommand(getInitCommand())
	rootCmd.AddCommand(getPrepareCommand())
	rootCmd.AddCommand(getReadPlanCommand())
	rootCmd.AddCommand(getReleaseCommand())

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Check that all paths are within the same repository
	for _, modulePath := range modulePaths {
		moduleRepo, err := gitshell.GitResolveRoot(modulePath)
		if err != nil {
			return fmt.Errorf("unable to determine repository root: %s\n%w", err)
		}

		if repoRoot == "" {
			repoRoot = moduleRepo
		} else if repoRoot != moduleRepo {
			return errors.New("all paths have to be in the same repository")
		}
	}

	if repoRoot == "" {
		return errors.New("no path specified")
	}

	configPath := fmt.Sprintf("%s/.kaeter.config.yaml", repoRoot)
	v.SetConfigFile(configPath)

	// Attempt to parse the config file, ignore if we fail to do so
	_ = v.ReadInConfig()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if entry, ok := configMap[f.Name]; ok && !f.Changed && v.IsSet(entry) {
			val := v.GetString(entry)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

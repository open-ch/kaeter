package cmd

import (
	"fmt"
	"os"

	"github.com/open-ch/go-libs/gitshell"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/spf13/pflag"
)

const versionsFileNameRegex = `versions\.ya?ml`
const makeFile = "Makefile"
var configMap = map[string]string{
	"git-main-branch": "git.main.branch",
}

var (
	// Points to the module to be released
	modulePath string
	gitMainBranch string

	rootCmd = &cobra.Command{
		Use:   "kaeter",
		Short: "kaeter handles the releasing and versioning of your modules within a fat repo.",
		Long: `kaeter offers a standard approach for releasing and versioning arbitrary artifacts. 
Its goal is to provide a 'descriptive release' process, in which developers request the release of given artifacts, 
and upon acceptation of the request, a separate build infrastructure is in charge of carrying out the build.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			return initializeConfig(cmd)
		},
	}

	// Logger...
	logger = log.New()
)

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVarP(&modulePath, "path", "p", ".",
		`Path to where kaeter must work from. This is either the module for which a release is required,
or the repository for which a release plan must be executed.`)

	rootCmd.PersistentFlags().StringVar(&gitMainBranch, "git-main-branch", "origin/master",
		`Defines the main branch of the repository, can also be set in the configuration file as "git.main.branch".`)

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Add the repo root
	repoRoot := gitshell.GitResolveRoot(modulePath)
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

// Execute runs the whole enchilada, baby!
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

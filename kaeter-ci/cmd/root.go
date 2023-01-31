package cmd

import (
	"github.com/open-ch/go-libs/gitshell"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	repoPath string
	logger   = log.New()
)

// Execute runs the tool
func Execute() {
	var rawLogLevel, path string

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})

	rootCmd := &cobra.Command{
		Use:   "kaeter-ci",
		Short: "Extraction of information on kaeter modules for use in CI pipelines",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logLevel, err := log.ParseLevel(rawLogLevel)
			if err != nil {
				logger.Fatalln(err)
			}
			logger.Level = logLevel

			if path == "" {
				_ = cmd.Usage() // Print usage/help text of command
				logger.Fatalf("path is a required variable")
			}

			repoPath, err = gitshell.GitResolveRoot(path)
			if err != nil {
				logger.Fatalf("unable to determine repository root from '%s': %s\n%s", path, repoPath, err)
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&path, "path", "p", "", "path of the repository or module")
	rootCmd.PersistentFlags().StringVarP(&rawLogLevel, "log-level", "l", "info", "log level (info, debug, trace, ...)")

	_ = rootCmd.MarkFlagRequired("path")

	rootCmd.AddCommand(getCheckCommand())
	rootCmd.AddCommand(getDetectAllCommand())
	rootCmd.AddCommand(getModulesCommand())

	if err := rootCmd.Execute(); err != nil {
		logger.Fatalln(err)
	}
}

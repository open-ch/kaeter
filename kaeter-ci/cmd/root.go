package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// path of a module or repository (will be resovled to the git root later anyways)
	path   string
	logger = log.New()
)

// Execute runs the tool
func Execute() {
	var rawLogLevel string

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})

	rootCmd := &cobra.Command{
		Use:   "kaeter-ci",
		Short: "kaeter-ci speeds up the CI by preparing change information.",
		Long: `kaeter-police examine the diff and outputs extended information about which
Bazel targets where change or Kaeter modules touched.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logLevel, err := log.ParseLevel(rawLogLevel)
			if err != nil {
				logger.Fatal(err)
			}
			logger.Level = logLevel
		},
	}

	rootCmd.PersistentFlags().StringVarP(&path, "path", "p", ".",
		`The path to the repository`)
	rootCmd.PersistentFlags().StringVarP(&rawLogLevel, "log-level", "l", "info",
		`Log level can be any of info, debug, trace`)

	rootCmd.AddCommand(getModulesCommand())
	rootCmd.AddCommand(getCheckCommand())

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

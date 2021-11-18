package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Points to the module to be checked
	path string
	logLevel = "info"

	rootCmd = &cobra.Command{
		Use:   "kaeter-ci",
		Short: "kaeter-ci speeds up the CI by preparing change information.",
		Long: `kaeter-police examine the diff and outputs extended information about which
Bazel targets where change or Kaeter modules touched.`,
	}

	logger = log.New()
)

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVarP(&path, "path", "p",".",
		`The path to the repository`)

	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level","l","info",
		`Log level can be any of info, debug, trace`)

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}

// Execute runs the tool
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

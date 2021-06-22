package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// to ve matched against a filename (ie, not and entire path)
const versionsFileRegex = `^versions.ya?ml`
const readmeFile = "README.md"
const changelogFile = "CHANGELOG.md"

var (
	// Points to the module to be checked
	rootPath string

	rootCmd = &cobra.Command{
		Use:   "kaeter-police",
		Short: "kaeter-police makes sure that the basic quality requirements for packages are met.",
		Long: `kaeter-police examines all the packages that are managed with kaeter and it prevents their releases,
if not all quality criteria are met.
The goal is to make sure that the packages are easy to use, maintain and improve.`,
	}

	logger = log.New()
)

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVarP(&rootPath, "path", "p", ".",
		`Path to where kaeter-police must work from.`)

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

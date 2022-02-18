package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Points to the root folder from which to check
	rootPath string
)

// Execute runs the tool
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "kaeter-police",
		Short: "kaeter-police makes sure that the basic quality requirements for packages are met.",
		Long: `kaeter-police examines all the packages that are managed with kaeter and
it prevents their releases if not all quality criteria are met.
The goal is to make sure that the packages are easy to use, maintain and improve.`,
	}

	rootCmd.PersistentFlags().StringVarP(&rootPath, "path", "p", "",
		`Path where kaeter-police starts from.`)
	rootCmd.MarkPersistentFlagRequired("path")

	rootCmd.AddCommand(getCheckCommand())

	log.SetPrefix("kaeter-police: ")
	log.SetFlags(0)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

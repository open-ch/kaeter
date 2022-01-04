package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var (
	rootCmd = &cobra.Command{
		Use:   "run",
		Short: "Basic quality checks for the specified module.",
	}
)

func init() {
	cobra.OnInitialize()
}

// Execute runs the tool
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

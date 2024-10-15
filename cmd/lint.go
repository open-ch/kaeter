package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/lint"
)

func getLintCommand() *cobra.Command {
	var strict bool
	command := &cobra.Command{
		Use:   "lint",
		Short: "Basic quality checks for the detected modules.",
		Long: `Then detects Kaeter modules starting from the given path,
for every kaeter-managed module (which has a versions.yaml file) the following
are checked:
- the existence of README.md
- the existence of a changelog (defaults to CHANGELOG.md)
- the changelog is up-to-date with versions.yaml (for releases)
- the dependencies listed in versions.yaml are existing paths
- the detected kaeter Makefile contains valid required targets

Strict only checks:
- the module has no pending/dangling autorelease

on error it will include details about all issues detected in all the scanned modules.`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, _ []string) error {
			// TODO allow using --path instead of repoRoot to lint subset
			repositoryRoot := viper.GetString("repoRoot")
			if strict {
				log.Info("Linting in strict mode.")
			}
			err := lint.CheckModulesStartingFrom(lint.CheckConfig{
				RepoRoot: repositoryRoot,
				Strict:   strict,
			})
			if err != nil {
				return fmt.Errorf("lint failed: %w", err)
			}
			log.Info("No issues detected.")
			return nil
		},
	}

	command.Flags().BoolVar(&strict, "strict", false, "Enable additional strict checks when validating modules")

	return command
}

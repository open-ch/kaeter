package cmd

import (
	"encoding/json"
	"os"
	kaeterChange "github.com/open-ch/kaeter/kaeter/change"
	"github.com/open-ch/kaeter/kaeter/modules"
	"path/filepath"

	"github.com/spf13/cobra"
)

func getCheckCommand() *cobra.Command {
	var currentCommit string
	var previousCommit string
	var pullRequest kaeterChange.PullRequest
	var outputFile string

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Checks everything that has changed between two commits",
		Long: `Provides metadata about changes between two commits, including:
 - Helm Charts
 - Kaeter modules
 - Commit info and details
 - Optional pull request info (must be passed in)
The output will be written to a json file.
`,
		Run: func(cmd *cobra.Command, args []string) {
			kaeterModules, err := modules.GetKaeterModules(repoPath)
			if err != nil {
				logger.Fatalf("failed to detect kaeter modules: %s", err)
			}

			detector := &kaeterChange.Detector{
				Logger:         logger,
				RootPath:       repoPath,
				PreviousCommit: previousCommit,
				CurrentCommit:  currentCommit,
				KaeterModules:  kaeterModules,
				PullRequest:    &pullRequest,
			}

			err = runChangeDetection(detector, outputFile)
			if err != nil {
				logger.Fatalf("Change detection failed: %s", err)
			}
		},
	}

	flags := checkCmd.Flags()
	flags.StringVar(&previousCommit, "previous-commit", "HEAD~1", "The previous commit")
	flags.StringVar(&currentCommit, "latest-commit", "HEAD", "The current commit")
	flags.StringVar(&pullRequest.Title, "pr-title", "", "Optional: if a pull request is open, the title")
	flags.StringVar(&pullRequest.Body, "pr-body", "", "Optional: if a pull request is open, the body")
	flags.StringVar(&outputFile, "output", "./changeset.json", "The path to the file containing the change information")
	return checkCmd
}

func runChangeDetection(detector *kaeterChange.Detector, outputFile string) error {
	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(repoPath, outputFile)
	}

	changeset, err := detector.Check()
	if err != nil {
		return err
	}

	changesetJSON, err := json.MarshalIndent(changeset, "", "    ")
	if err != nil {
		return err
	}
	detector.Logger.Infoln(string(changesetJSON))
	return os.WriteFile(outputFile, changesetJSON, 0600)
}

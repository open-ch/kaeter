package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	kaeterChange "github.com/open-ch/kaeter/kaeter-ci/pkg/change"
	"path/filepath"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/spf13/cobra"
)

func getCheckCommand() *cobra.Command {
	var currentCommit string
	var previousCommit string
	var outputFile string
	var skipBazelCheck bool

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Checks everything that has changed between two commits",
		Long: `Provides some metadata about what has changed between two commits. This can be about:
 - Helm Charts
 - Kaeter modules
 - Bazel Targets
 - Commit info and details

The output will be written to a json file.
`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runCheck(previousCommit, currentCommit, outputFile, skipBazelCheck)
			if err != nil {
				logger.Fatalf("Check failed: %s", err)
			}
		},
	}

	checkCmd.PersistentFlags().StringVar(&previousCommit, "previous-commit", "HEAD~1",
		`The previous commit `)
	checkCmd.PersistentFlags().StringVar(&currentCommit, "latest-commit", "HEAD",
		`The current commit`)
	checkCmd.PersistentFlags().StringVar(&outputFile, "output", "./changeset.json",
		`The path to the file containing the change information`)
	checkCmd.PersistentFlags().BoolVar(&skipBazelCheck, "skip-bazel", false,
		`Skip the check for bazel changes`)

	return checkCmd
}

func runCheck(previousCommit, currentCommit, outputFile string, skipBazelCheck bool) error {
	rootPath, err := gitshell.GitResolveRoot(path)
	if err != nil {
		return fmt.Errorf("unable to determine repository root: %s\n%w", err)
	}

	detector := &kaeterChange.Detector{logger, rootPath, previousCommit, currentCommit}

	info := detector.Check(skipBazelCheck)

	changesetJSON, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		return err
	}
	logger.Info(string(changesetJSON))

	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(path, outputFile)
	}

	return ioutil.WriteFile(outputFile, changesetJSON, 444)
}

package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	kaeterChange "github.com/open-ch/kaeter/kaeter-ci/pkg/change"
	"path/filepath"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {

	var currentCommit string
	var previousCommit string
	var outputFile string

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Checks everything that has changed between two commits",
		Long: `Provides some metadata about what has changed between two commits. This can be about:
 - Helm Charts
 - Kaeter modules
 - Bazel Targets

The output will be written to a json file.
`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runCheck(previousCommit, currentCommit, outputFile)
			if err != nil {
				logger.Errorf("Check failed: %s", err)
				os.Exit(1)
			}
		},
	}

	checkCmd.PersistentFlags().StringVar(&previousCommit, "previous-commit", "HEAD~1",
		`The previous commit `)

	checkCmd.PersistentFlags().StringVar(&currentCommit, "latest-commit", "HEAD",
		`The current commit`)

	checkCmd.PersistentFlags().StringVar(&outputFile, "output", "./target.json",
		`The path to the file containing the change information`)

	rootCmd.AddCommand(checkCmd)
}

func runCheck(previousCommit, currentCommit, outputFile string) error {
	rootPath := gitshell.GitResolveRoot(path)
	ll, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	detector := kaeterChange.New(ll, rootPath, previousCommit, currentCommit)

	info := detector.Check()

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

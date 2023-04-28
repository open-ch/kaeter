package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/open-ch/kaeter/kaeter/actions"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// planStatus tells us whether we found a release plan or not.
// it will also be used as the return code of the command.
type planStatus int

const (
	foundPlan      planStatus = iota // 0
	repoError                        // 1
	noPlanInCommit                   // 2
)

func getReadPlanCommand() *cobra.Command {
	var jsonOutputPath string
	var commitMessage string

	cmd := &cobra.Command{
		Use:   "read-plan",
		Short: "Attempts to read a release plan from the last commit",
		Long: `Attempts to read a release plan from the last commit, displaying it's content to stdout if one could be found,
and returning an error status if no plan was detected.

Path doesn't need to be to a specific module, it can be to the repo itself.

Useful for using as part of a conditional pipeline check.`,
		PreRunE: validateAllPathFlags,
		Run: func(_ *cobra.Command, args []string) {
			retCode, err := readReleasePlan(logger, repoRoot, jsonOutputPath, commitMessage)
			if err != nil {
				logger.Errorf("read: %s", err)
			}
			os.Exit(int(retCode))
		},
	}

	cmd.Flags().StringVar(&jsonOutputPath, "json-output", "", "If provided the plan will be written to that path")
	cmd.Flags().StringVar(&commitMessage, "commit-message", "", "Read release plan from this string instead of git")

	return cmd
}

// readReleasePlan attempts to read a release plan from the last commit, displaying its content if found.
// Returns a return code of 0 if a plan was found, and 2 if not.
// Optionally outputs a machine readable plan in json at the given path
func readReleasePlan(logger *logrus.Logger, repoRoot, jsonOutputPath, commitMessage string) (planStatus, error) {
	if commitMessage == "" {
		logger.Debugln("no commit message passed in, attempting to read from HEAD with git")
		headCommitMessage, err := getHeadCommitMessage(repoRoot)
		if err != nil {
			return repoError, err
		}
		commitMessage = headCommitMessage
	}

	// Before trying to read a plan, we use the check method which is a bit more stringent.
	logger.Debugf("reading release plan from: \n%s", commitMessage)
	if actions.HasReleasePlan(commitMessage) {
		rp, err := actions.ReleasePlanFromCommitMessage(commitMessage)
		if err != nil {
			return repoError, fmt.Errorf("failed to read release plan from commit message: %w", err)
		}
		logger.Infof("Found release plan with release targets:")
		for _, target := range rp.Releases {
			logger.Infof("\t%s", target.Marshal())
		}

		if jsonOutputPath != "" {
			releasesJSON, err := json.Marshal(rp.Releases)
			if err != nil {
				return repoError, err
			}
			err = os.WriteFile(jsonOutputPath, []byte(releasesJSON), 0644)
			if err != nil {
				return repoError, err
			}
			logger.Debugf("release plan written to: %s", jsonOutputPath)
		}

		return foundPlan, nil
	}
	logger.Infof("The current HEAD commit does not seem to contain a release plan.")
	return noPlanInCommit, nil
}

func getHeadCommitMessage(repoRoot string) (string, error) {
	headCommitMessage, err := gitshell.GitCommitMessageFromHash(repoRoot, "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit message for HEAD: %w", err)
	}
	return headCommitMessage, nil
}

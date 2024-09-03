package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
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
		Run: func(_ *cobra.Command, _ []string) {
			repositoryRoot := viper.GetString("repoRoot")
			retCode, err := readReleasePlan(repositoryRoot, jsonOutputPath, commitMessage)
			if err != nil {
				log.Errorf("read: %s", err)
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
func readReleasePlan(repoRoot, jsonOutputPath, commitMessage string) (planStatus, error) {
	if commitMessage == "" {
		log.Debug("no commit message passed in, attempting to read from HEAD with git")
		headCommitMessage, err := getHeadCommitMessage(repoRoot)
		if err != nil {
			return repoError, err
		}
		commitMessage = headCommitMessage
	}

	// Before trying to read a plan, we use the check method which is a bit more stringent.
	log.Debug("reading release plan from", "commitMessage", commitMessage)
	if !actions.HasReleasePlan(commitMessage) {
		log.Info("The current HEAD commit does not seem to contain a release plan.")
		return noPlanInCommit, nil
	}

	rp, err := actions.ReleasePlanFromCommitMessage(commitMessage)
	if err != nil {
		return repoError, fmt.Errorf("failed to read release plan from commit message: %w", err)
	}
	log.Info("Found release plan with release targets:")
	for _, target := range rp.Releases {
		log.Infof("\t%s", target.Marshal())
	}

	if jsonOutputPath != "" {
		releasesJSON, err := json.Marshal(rp.Releases)
		if err != nil {
			return repoError, err
		}
		err = os.WriteFile(jsonOutputPath, releasesJSON, 0600)
		if err != nil {
			return repoError, err
		}
		log.Debug("release plan written to", "outputPath", jsonOutputPath)
	}

	return foundPlan, nil
}

func getHeadCommitMessage(repoRoot string) (string, error) {
	headCommitMessage, err := git.GetCommitMessageFromRef(repoRoot, "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit message for HEAD: %w", err)
	}
	return headCommitMessage, nil
}

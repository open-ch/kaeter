package cmd

import (
	"encoding/json"
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

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

	readPlanCmd := &cobra.Command{
		Use:   "read-plan",
		Short: "Attempts to read a release plan from the last commit",
		Long: `Attempts to read a release plan from the last commit, displaying it's content to stdout if one could be found,
and returning an error status if no plan was detected.

Useful for using as part of a conditional pipeline check.'`,
		Run: func(cmd *cobra.Command, args []string) {
			retCode, err := readReleasePlan(logger, repoRoot, jsonOutputPath)
			if err != nil {
				logger.Errorf("read: %s", err)
				os.Exit(int(retCode))
			}
			os.Exit(int(retCode))
		},
	}

	readPlanCmd.Flags().StringVar(&jsonOutputPath, "json-output", "", "If provided the plan will be written to that path")

	return readPlanCmd
}

// readReleasePlan attempts to read a release plan from the last commit, displaying its content if found.
// Returns a return code of 0 if a plan was found, and 2 if not.
// Optionally outputs a machine readable plan in json at the given path
func readReleasePlan(logger *logrus.Logger, repoRoot string, jsonOutputPath string) (planStatus, error) {

	headHash := gitshell.GitResolveRevision(repoRoot, "HEAD")
	headCommitMessage, err := gitshell.GitCommitMessageFromHash(repoRoot, headHash)

	if err != nil {
		logger.Errorf("Failed to get commit message for HEAD: %s", err)
		return repoError, err
	}

	// Before trying to read a plan, we use the check method which is a bit more stringent.
	if kaeter.HasReleasePlan(headCommitMessage) {
		rp, err := kaeter.ReleasePlanFromCommitMessage(headCommitMessage)
		if err != nil {
			logger.Errorf("Failed to read release plan from head commit!")
			return repoError, err
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

package cmd

import (
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/open-ch/go-libs/gitshell"
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
	readPlanCmd := &cobra.Command{
		Use:   "read-plan",
		Short: "Attempts to read a release plan from the last commit",
		Long: `Attempts to read a release plan from the last commit, displaying it's content to stdout if one could be found,
and returning an error status if no plan was detected.

Useful for using as part of a conditional pipeline check.'`,
		Run: func(cmd *cobra.Command, args []string) {
			retCode, err := runReadPlan(repoRoot)
			if err != nil {
				logger.Errorf("read: %s", err)
				os.Exit(1)
			}
			os.Exit(int(retCode))
		},
	}

	return readPlanCmd
}

// runReadPlan attempts to read a release plan from the last commit, displaying its content if found.
// Returns a return code of 0 if a plan was found, and 2 if not.
func runReadPlan(modulePath string) (planStatus, error) {

	headHash := gitshell.GitResolveRevision(modulePath, "HEAD")

	headCommitMessage := gitshell.GitCommitMessageFromHash(modulePath, headHash)

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
		return foundPlan, nil
	}
	logger.Infof("The current HEAD commit does not seem to contain a release plan.")
	return noPlanInCommit, nil
}

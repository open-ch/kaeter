package git

import (
	"fmt"
	"regexp"
)

// ValidateCommitIsOnTrunk can be used to validate that a given hash
// has a common ancestry with a specific branch.
func ValidateCommitIsOnTrunk(modulePath, trunkBranch, commitHash string) error {
	branchPattern := fmt.Sprintf("*%s*", trunkBranch)
	// We use the branch pattern to avoid listing all branches this allows
	// CI to fetch only trunk before running kaeter.
	output, err := BranchContains(modulePath, commitHash, branchPattern)
	if err != nil {
		return fmt.Errorf("unable to fetch %s before checking commit: \n%s\n%w", trunkBranch, output, err)
	}
	// Check if trunkBranch or remotes/origin/trunkBranch is part of the list of branches
	// Example output:
	// ```
	// * HEAD detached ...
	//   trunkBranch
	//   remotes/origin/trunkBranch
	// ```
	// So we look for:
	// - Start of a line with star or space (`^[* ] `)
	// - Optional remote match (`(?:remotes/origin/)?`)
	// - The repository's configured trunk as possed in at the end of the line (`%s$`)
	// (the remote (origin) could be made configurable)
	expectedBranchRegex := regexp.MustCompile(fmt.Sprintf("(?m)^[* ] (?:remotes/origin/)?%s$", regexp.QuoteMeta(trunkBranch)))
	if !expectedBranchRegex.MatchString(output) {
		return fmt.Errorf("commit (%s) not on trunk branch (%s): \n%s", commitHash, trunkBranch, output)
	}

	return nil
}

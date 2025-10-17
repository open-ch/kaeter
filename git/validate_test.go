package git

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCommitIsOnTrunk(t *testing.T) {
	var tests = []struct {
		name     string
		commit   string
		hasError bool
	}{
		{
			name:     "Commit hash on trunk",
			commit:   "firstCommit",
			hasError: false,
		},
		{
			name:     "Non existent commit hash",
			commit:   "4793ffbf3c9312f801ed322735781151790e5932",
			hasError: true,
		},
		{
			name:     "Commit hash not on trunk",
			commit:   "branchCommit",
			hasError: true,
		},
	}
	defaultTrunkBranch := "main"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testRepoFolder := createMockRepo(t)
			firstCommit := commitFileAndGetHash(t, testRepoFolder, "README.md", "# unit testing", "init test repo")
			t.Logf("firstCommit hash: '%s'", firstCommit)
			gitExec(t, testRepoFolder, "switch", "-c", "anotherbranch")
			branchCommit := commitFileAndGetHash(t, testRepoFolder, "main.go", "// Empty file", "commit on a branch")
			t.Logf("branchCommit hash: '%s'", branchCommit)
			commitToCheck := tc.commit
			// Allow picking commits dynamically based on name:
			if tc.commit == "firstCommit" {
				commitToCheck = firstCommit
			}
			if tc.commit == "branchCommit" {
				commitToCheck = branchCommit
			}
			t.Logf("Checking for hash: '%s'", strings.TrimSpace(commitToCheck))

			err := ValidateCommitIsOnTrunk(testRepoFolder, defaultTrunkBranch, commitToCheck)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

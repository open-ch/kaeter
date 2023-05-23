package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffNameStatus(t *testing.T) {
	var tests = []struct {
		name            string
		previousCommit  string
		currentCommit   string
		expectedChanges map[string]FileChangeStatus
	}{
		{
			name:            "Empty Diff",
			previousCommit:  "HEAD~3",
			currentCommit:   "HEAD~2",
			expectedChanges: map[string]FileChangeStatus{},
		},
		{
			name:           "Diff with changes",
			previousCommit: "HEAD~1",
			currentCommit:  "HEAD",
			expectedChanges: map[string]FileChangeStatus{
				"modifiedFile": Modified,
				"addedFile":    Added,
				"deletedFile":  Deleted,
				// Because we use --no-renames a renamed file counts as both added and deleted
				// We should add support for renames but it would be difficult to keep it
				// backwards compatible and renames with too many changes eventually add/remove
				// so behavior would not always be 100% consistent.
				"renamedFile": Added,
				"namedFile":   Deleted,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testRepoFolder := mockGitRepoCommitsToDiff(t)
			defer os.RemoveAll(testRepoFolder)
			t.Logf("Temp test folder: %s\n(disable `defer os.RemoveAll(testRepoFolder)` to keep for debugging)", testRepoFolder)

			results, err := DiffNameStatus(testRepoFolder, tc.previousCommit, tc.currentCommit)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedChanges, results)
		})
	}
}

func mockGitRepoCommitsToDiff(t *testing.T) string {
	t.Helper()

	repoPath := createMockRepo(t)

	gitExec(t, repoPath, "commit", "--allow-empty", "-m", "c1")
	gitExec(t, repoPath, "commit", "--allow-empty", "-m", "c2")

	addFileToRepo(t, repoPath, "deletedFile", "")
	addFileToRepo(t, repoPath, "modifiedFile", "")
	addFileToRepo(t, repoPath, "namedFile", "renamed")
	gitExec(t, repoPath, "add", ".")
	gitExec(t, repoPath, "commit", "-m", "c3")

	deleteFileFromRepo(t, repoPath, "deletedFile")
	deleteFileFromRepo(t, repoPath, "namedFile")
	addFileToRepo(t, repoPath, "modifiedFile", "modified")
	addFileToRepo(t, repoPath, "addedFile", "")
	addFileToRepo(t, repoPath, "renamedFile", "renamed")
	gitExec(t, repoPath, "add", ".")
	gitExec(t, repoPath, "commit", "-m", "c4")

	return repoPath
}

package git

import (
	"os"
	"path/filepath"

	"testing"

	"github.com/stretchr/testify/assert"
)

func createMockRepo(t *testing.T) string {
	t.Helper()
	testFolder, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	// Our gitshell library doesn't have init or config so we do it inline here
	gitExec(t, testFolder, "init")

	// Set local user on the tmp repo, to avoid errors when git commit finds no author
	gitExec(t, testFolder, "config", "user.email", "unittest@example.ch")
	gitExec(t, testFolder, "config", "user.name", "Unit Test")

	// Note:
	// The build agents currently have an older version of git and don't supprot
	// Renaming the branch, so the default new repo branch is master.
	// It's possible it randomly changes to main once we update one day and this
	// tests starts failing.
	// However atempts to change to a deterministic branch (i.e. test)
	// consistently failed to run on CI
	// git init --initial-branch test -> not supported on older git versions
	// git branch -M test -> fails to rename the branch

	return testFolder
}

func commitFileAndGetHash(t *testing.T, repoPath, filename, fileContent, commitMessage string) string {
	t.Helper()
	err := os.WriteFile(filepath.Join(repoPath, filename), []byte(fileContent), 0600)
	assert.NoError(t, err)

	gitExec(t, repoPath, "add", ".")
	gitExec(t, repoPath, "commit", "-m", commitMessage)

	hash, err := ResolveRevision(repoPath, "HEAD")
	assert.NoError(t, err)

	return hash
}

func gitExec(t *testing.T, repoPath, subCommand string, additionalArgs ...string) {
	t.Helper()
	output, err := git(repoPath, subCommand, additionalArgs...)
	t.Log(string(output))
	assert.NoError(t, err)
}

package mocks

import (
	"os"
	"os/exec"
	"path/filepath"

	"testing"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/stretchr/testify/assert"
)

const EmptyMakefileContent = ".PHONY: build test snapshot release"
const EmptyVersionsYAML = `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`

// CreateMockKaeterRepo is a test helper to create a mock kaeter module in a tmp fodler
// it returns the path to the tmp folder. Caller is responsible for deleting it.
func CreateMockKaeterRepo(t *testing.T, makefileContent, commitMessage, versionsYAML string) string {
	t.Helper()
	testFolder := CreateMockRepo(t)

	CreateMockFile(t, testFolder, "Makefile", makefileContent)
	CreateMockFile(t, testFolder, "versions.yaml", versionsYAML)
	_, err := gitshell.GitAdd(testFolder, ".")
	assert.NoError(t, err)
	_, err = gitshell.GitCommit(testFolder, commitMessage)
	assert.NoError(t, err)

	return testFolder
}

func CreateMockRepo(t *testing.T) string {
	t.Helper()
	testFolder := CreateTmpFolder(t)

	// Our gitshell library doesn't have init or config so we do it inline here
	execGitCommand(t, testFolder, "init")

	// Set local user on the tmp repo, to avoid errors when git commit finds no author
	execGitCommand(t, testFolder, "config", "user.email", "unittest@example.ch")
	execGitCommand(t, testFolder, "config", "user.name", "Unit Test")

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

func CommitFileAndGetHash(t *testing.T, repoPath, filename, fileContent, commitMessage string) string {
	t.Helper()
	CreateMockFile(t, repoPath, filename, fileContent)

	// Note these don't return errors they'll just exit on failure:
	_, err := gitshell.GitAdd(repoPath, ".")
	assert.NoError(t, err)
	_, err = gitshell.GitCommit(repoPath, commitMessage)
	assert.NoError(t, err)
	hash, err := gitshell.GitResolveRevision(repoPath, "HEAD")
	assert.NoError(t, err)

	return hash
}

func SwitchToNewBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()

	execGitCommand(t, repoPath, "switch", "-c", branchName)
}

func execGitCommand(t *testing.T, repoPath string, additionalArgs ...string) {
	t.Helper()

	gitCmd := exec.Command("git", additionalArgs...)
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	t.Log(string(output))
	assert.NoError(t, err)
}

func CreateTmpFolder(t *testing.T) string {
	t.Helper()
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	return testFolderPath
}

func CreateMockFile(t *testing.T, tmpPath, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(tmpPath, filename), []byte(content), 0644)
	assert.NoError(t, err)
}

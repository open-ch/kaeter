package kaeter

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"testing"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/stretchr/testify/assert"
)

func createMockKaeterRepo(t *testing.T, makefileContent string, commitMessage string, versionsYAML string) string {
	t.Helper()
	testFolder := createMockRepo(t)

	createMockFile(t, testFolder, "Makefile", makefileContent)
	createMockFile(t, testFolder, "versions.yaml", versionsYAML)
	_, err := gitshell.GitAdd(testFolder, ".")
	assert.NoError(t, err)
	_, err = gitshell.GitCommit(testFolder, commitMessage)
	assert.NoError(t, err)

	return testFolder
}

func createMockRepo(t *testing.T) string {
	t.Helper()
	testFolder := createTmpFolder(t)

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
	// However attemps to change to a deterministic branch (i.e. test)
	// consistently failed to run on CI
	// git init --initial-branch test -> not supported on older git versions
	// git branch -M test -> fails to rename the branch

	return testFolder
}

func commitFileAndGetHash(t *testing.T, repoPath, filename, fileContent, commitMessage string) string {
	t.Helper()
	createMockFile(t, repoPath, filename, fileContent)

	// Note these don't return errors they'll just exit on failure:
	_, err := gitshell.GitAdd(repoPath, ".")
	assert.NoError(t, err)
	_, err = gitshell.GitCommit(repoPath, commitMessage)
	assert.NoError(t, err)
	hash, err := gitshell.GitResolveRevision(repoPath, "HEAD")
	assert.NoError(t, err)

	return hash
}

func switchToNewBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()

	execGitCommand(t, repoPath, "switch", "-c", branchName)
}

func execGitCommand(t *testing.T, repoPath string, additionalArgs ...string) {
	t.Helper()

	gitCmd := exec.Command("git", additionalArgs...)
	gitCmd.Dir = repoPath
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	err := gitCmd.Run()
	assert.NoError(t, err)
}

func createTmpFolder(t *testing.T) string {
	t.Helper()
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	return testFolderPath
}

func createMockFile(t *testing.T, tmpPath string, filename string, content string) {
	t.Helper()
	err := ioutil.WriteFile(filepath.Join(tmpPath, filename), []byte(content), 0644)
	assert.NoError(t, err)
}

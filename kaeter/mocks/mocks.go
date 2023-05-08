package mocks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"testing"

	"github.com/stretchr/testify/assert"
)

// EmptyMakefileContent is the content of the minimal Makefile, used for testing
const EmptyMakefileContent = ".PHONY: build test snapshot release"

// EmptyVersionsYAML is the content of a minimal kaeter versions file, used for testing
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
	execGitCommand(t, testFolder, "add", ".")
	execGitCommand(t, testFolder, "commit", "-m", commitMessage)

	return testFolder
}

// AddSubDirKaeterMock is a test helper to create a mock kaeter module in a tmp fodler
// it returns the path to the tmp folder. Caller is responsible for deleting it.
func AddSubDirKaeterMock(t *testing.T, testFolder, modulePath, versionsYAML string) string {
	t.Helper()

	absPath := filepath.Join(testFolder, modulePath)
	err := os.Mkdir(absPath, 0755)
	assert.NoError(t, err)

	CreateMockFile(t, absPath, "Makefile", EmptyMakefileContent)
	CreateMockFile(t, absPath, "versions.yaml", versionsYAML)
	execGitCommand(t, testFolder, "add", ".")
	execGitCommand(t, testFolder, "commit", "-m", fmt.Sprintf("Add module %s", modulePath))

	return absPath
}

// CreateMockRepo initializes a mock git repository in a tmp folder
func CreateMockRepo(t *testing.T) string {
	t.Helper()
	testFolder := CreateTmpFolder(t)

	// Our git wrapper doesn't have init or config so we do it inline here
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

// CommitFileAndGetHash wrapper around git add and git commit, returns the hash of commit.
func CommitFileAndGetHash(t *testing.T, repoPath, filename, fileContent, commitMessage string) string {
	t.Helper()
	CreateMockFile(t, repoPath, filename, fileContent)

	execGitCommand(t, repoPath, "add", ".")
	execGitCommand(t, repoPath, "commit", "-m", commitMessage)
	gitCmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	t.Log(string(output))
	assert.NoError(t, err)
	return strings.TrimSpace(string(output))
}

// SwitchToNewBranch wrapper around git switch -c branchName
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

// CreateTmpFolder returns path to new temp folder for testing
func CreateTmpFolder(t *testing.T) string {
	t.Helper()
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	return testFolderPath
}

// CreateMockFile creates file with content in a tmp folder
func CreateMockFile(t *testing.T, tmpPath, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(tmpPath, filename), []byte(content), 0600)
	assert.NoError(t, err)
}

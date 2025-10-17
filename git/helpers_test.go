package git

import (
	"os"
	"path/filepath"

	"testing"

	"github.com/stretchr/testify/assert"
)

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

func createMockRepo(t *testing.T) string {
	t.Helper()
	testFolder := t.TempDir()

	// Our git library doesn't have init or config so we do it inline here
	gitExec(t, testFolder, "init", "--initial-branch=main")

	// Set local user on the tmp repo, to avoid errors when git commit finds no author
	gitExec(t, testFolder, "config", "user.email", "unittest@example.ch")
	gitExec(t, testFolder, "config", "user.name", "Unit Test")

	return testFolder
}

func addFileToRepo(t *testing.T, repoPath, filename, fileContent string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(repoPath, filename), []byte(fileContent), 0600)
	assert.NoError(t, err)
}

func deleteFileFromRepo(t *testing.T, repoPath, filename string) {
	t.Helper()
	fileToRemove := filepath.Join(repoPath, filename)
	if repoPath == "" || filename == "" || len(fileToRemove) < 5 {
		assert.Fail(t, "Invalid helper argumetns for deleting a file")
	}
	err := os.Remove(fileToRemove)
	assert.NoError(t, err)
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
	t.Log(output)
	assert.NoError(t, err)
}

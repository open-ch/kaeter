package change

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"testing"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/stretchr/testify/assert"
)

func createMockRepo(t *testing.T) string {
	t.Helper()
	testFolder := createTmpFolder(t)

	// Our gitshell library doesn't have init or config, so we run these the hard way
	gitInitCommand := exec.Command("git", "init")
	gitInitCommand.Dir = testFolder
	err := gitInitCommand.Run()
	assert.NoError(t, err)

	// Set local user on the tmp repo, to avoid errors when git commit finds no author
	gitConfigEmailCmd := exec.Command("git", "config", "user.email", "unittest@example.ch")
	gitConfigEmailCmd.Dir = testFolder
	gitConfigEmailCmd.Stdout = os.Stdout
	gitConfigEmailCmd.Stderr = os.Stderr
	err = gitConfigEmailCmd.Run()
	assert.NoError(t, err)

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

package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTreesExist(t *testing.T) {
	repoPath := createMockRepo(t)
	assert.NoError(t, os.MkdirAll(filepath.Join(repoPath, "a", "b"), 0o750))
	addFileToRepo(t, repoPath, "a/b/file", "")
	addFileToRepo(t, repoPath, "top", "")
	gitExec(t, repoPath, "add", ".")
	gitExec(t, repoPath, "commit", "-m", "c1")

	exists, err := TreesExist(repoPath, "HEAD", []string{"a", "a/b", "a/c", "top", "a/b/file"})

	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"a":        true,
		"a/b":      true,
		"a/c":      false,
		"top":      false, // a file, not a directory
		"a/b/file": false,
	}, exists)
}

func TestTreesExist_NoPaths(t *testing.T) {
	exists, err := TreesExist(t.TempDir(), "HEAD", nil)

	assert.NoError(t, err)
	assert.Empty(t, exists)
}

func TestTreesExist_UnknownCommit(t *testing.T) {
	repoPath := createMockRepo(t)
	gitExec(t, repoPath, "commit", "--allow-empty", "-m", "c1")

	exists, err := TreesExist(repoPath, "0000000000000000000000000000000000000000", []string{"a"})

	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{"a": false}, exists)
}

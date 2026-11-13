package change

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

func TestFiles_AllFiles(t *testing.T) {
	assert.Empty(t, (&Files{}).AllFiles(), "Expected empty slice for no files")

	files := Files{
		Added:    []string{"file1.txt", "file2.txt"},
		Modified: []string{"file3.txt"},
		Removed:  []string{"file4.txt"},
	}

	expected := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt"}
	allFiles := files.AllFiles()

	assert.Equal(t, expected, allFiles)
}

func TestFileCheck_Directories(t *testing.T) {
	repoPath, _ := mocks.CreateMockRepo(t)
	mocks.CreateMockFolder(t, repoPath, "existing/sub")
	mocks.CreateMockFolder(t, repoPath, "gone/deep")
	mocks.CreateMockFolder(t, repoPath, "shrinking/empty")
	mocks.CreateMockFile(t, repoPath, "existing/sub/file", "")
	mocks.CreateMockFile(t, repoPath, "gone/deep/file", "")
	mocks.CreateMockFile(t, repoPath, "shrinking/keep", "")
	mocks.CreateMockFile(t, repoPath, "shrinking/empty/file", "")
	previous := mocks.CommitFileAndGetHash(t, repoPath, "top", "", "previous")

	mocks.CreateMockFolder(t, repoPath, "existing/sub/new/deeper")
	mocks.CreateMockFolder(t, repoPath, "brand-new/a")
	mocks.CreateMockFile(t, repoPath, "existing/sub/new/deeper/file", "")
	mocks.CreateMockFile(t, repoPath, "existing/sub/file2", "")
	mocks.CreateMockFile(t, repoPath, "brand-new/a/file", "")
	assert.NoError(t, os.RemoveAll(filepath.Join(repoPath, "gone")))
	assert.NoError(t, os.RemoveAll(filepath.Join(repoPath, "shrinking", "empty")))
	current := mocks.CommitFileAndGetHash(t, repoPath, "top2", "", "current")

	detector := &Detector{RootPath: repoPath, PreviousCommit: previous, CurrentCommit: current}
	files, err := detector.FileCheck(&Information{})

	assert.NoError(t, err)
	assert.Equal(t, Directories{
		Added:   []string{"brand-new", "brand-new/a", "existing/sub/new", "existing/sub/new/deeper"},
		Removed: []string{"gone", "gone/deep", "shrinking/empty"},
	}, files.Directories)
}

func TestParentDirectories(t *testing.T) {
	assert.Equal(t,
		[]string{"a", "a/b", "c"},
		parentDirectories([]string{"a/b/file", "a/file", "c/file", "top"}))
	assert.Empty(t, parentDirectories(nil))
}

package kaeterpolice

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	existingFile                    = "CHANGELOG"
	existingFolder                  = "test-data"
	nonExistingFileInExistingFolder = "random"
	nonExistingFolder               = "any"
)

func TestCheckModulesStartingFromNoModules(t *testing.T) {
	repoPath := createMockRepoFolder(t)
	testPath := path.Join(repoPath, "test")
	defer os.RemoveAll(repoPath)

	err := CheckModulesStartingFrom(testPath)

	assert.NoError(t, err)
}

func TestCheckModulesStartingFromInvalidModules(t *testing.T) {
	repoPath := createMockRepoFolder(t)
	testPath := path.Join(repoPath, "test")
	err := os.WriteFile(path.Join(repoPath, "versions.yaml"), []byte("Hello, Gophers!"), 0655)

	defer os.RemoveAll(repoPath)

	err = CheckModulesStartingFrom(testPath)

	assert.Error(t, err)
}

func TestCheckExistenceRelative(t *testing.T) {
	// CHANGELOG exists
	err := checkExistence(existingFile, existingFolder)
	assert.NoError(t, err)

	// error due to the non-existence of the file "random" inside the existing folder test-data
	err = checkExistence(nonExistingFileInExistingFolder, existingFolder)
	errMsg := fmt.Sprintf(
		"Error in getting FileInfo about '%s': %s",
		nonExistingFileInExistingFolder,
		fmt.Sprintf("stat %s: no such file or directory", filepath.Join(existingFolder, nonExistingFileInExistingFolder)),
	)
	assert.EqualError(t, err, errMsg)

	// error due to the non-existence of the folder "any"
	err = checkExistence(existingFile, nonExistingFolder)
	errMsg = fmt.Sprintf(
		"Error in getting FileInfo about '%s': %s",
		nonExistingFolder,
		fmt.Sprintf("stat %s: no such file or directory", nonExistingFolder),
	)
	assert.EqualError(t, err, errMsg)
}

func TestCheckExistenceAbsolute(t *testing.T) {
	// getting absolute path for test-data
	abs, err := filepath.Abs(existingFolder)
	assert.NoError(t, err)

	// CHANGELOG exists
	err = checkExistence(existingFile, abs)
	assert.NoError(t, err)

	// error due to the non-existence of the file "random" inside the existing folder test-data
	err = checkExistence(nonExistingFileInExistingFolder, abs)
	errMsg := fmt.Sprintf(
		"Error in getting FileInfo about '%s': %s",
		nonExistingFileInExistingFolder,
		fmt.Sprintf("stat %s: no such file or directory", filepath.Join(abs, nonExistingFileInExistingFolder)),
	)
	assert.EqualError(t, err, errMsg)

	// getting absolute path for a non-existing folder
	abs, err = filepath.Abs(nonExistingFolder)
	assert.NoError(t, err)

	// error due to the non-existence of the folder "any"
	err = checkExistence(existingFile, abs)
	errMsg = fmt.Sprintf("Error in getting FileInfo about '%s': %s", abs, fmt.Sprintf("stat %s: no such file or directory", abs))
	assert.EqualError(t, err, errMsg)
}

func TestCheckChangelog(t *testing.T) {
	testDataPath, err := filepath.Abs(existingFolder)
	assert.NoError(t, err)
	versionsFilePath := path.Join(testDataPath, "dummy-versions-valid")
	changelogFilePath := path.Join(testDataPath, "dummy-changelog-SemVer")

	err = checkChangelog(versionsFilePath, changelogFilePath)

	assert.NoError(t, err)
}

func createMockRepoFolder(t *testing.T) (repoPath string) {
	repoPath, err := os.MkdirTemp("", "kaeter-police-*")
	assert.NoError(t, err)

	err = os.Mkdir(path.Join(repoPath, ".git"), 0755)
	assert.NoError(t, err)

	err = os.Mkdir(path.Join(repoPath, "test"), 0755)
	assert.NoError(t, err)

	return repoPath
}

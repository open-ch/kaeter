package cmd

import (
	"fmt"
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

func TestCheckExistenceRelative(t *testing.T) {
	// CHANGELOG exists
	err := checkExistence(existingFile, existingFolder)
	assert.NoError(t, err)

	// error due to the non-existence of the file "random" inside the existing folder test-data
	err = checkExistence(nonExistingFileInExistingFolder, existingFolder)
	errMsg := fmt.Sprintf("Error in getting FileInfo about '%s': %s", nonExistingFileInExistingFolder, fmt.Sprintf("stat %s: no such file or directory", filepath.Join(existingFolder, nonExistingFileInExistingFolder)))
	assert.EqualError(t, err, errMsg)

	// error due to the non-existence of the folder "any"
	err = checkExistence(existingFile, nonExistingFolder)
	errMsg = fmt.Sprintf("Error in getting FileInfo about '%s': %s", nonExistingFolder, fmt.Sprintf("stat %s: no such file or directory", nonExistingFolder))
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
	errMsg := fmt.Sprintf("Error in getting FileInfo about '%s': %s", nonExistingFileInExistingFolder, fmt.Sprintf("stat %s: no such file or directory", filepath.Join(abs, nonExistingFileInExistingFolder)))
	assert.EqualError(t, err, errMsg)

	// getting absolute path for a non-existing folder
	abs, err = filepath.Abs(nonExistingFolder)
	assert.NoError(t, err)

	// error due to the non-existence of the folder "any"
	err = checkExistence(existingFile, abs)
	errMsg = fmt.Sprintf("Error in getting FileInfo about '%s': %s", abs, fmt.Sprintf("stat %s: no such file or directory", abs))
	assert.EqualError(t, err, errMsg)
}

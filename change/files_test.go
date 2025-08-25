package change

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFiles_AllFiles(t *testing.T) {
	assert.Empty(t, Files{}.AllFiles(), "Expected empty slice for no files")

	files := Files{
		Added:    []string{"file1.txt", "file2.txt"},
		Modified: []string{"file3.txt"},
		Removed:  []string{"file4.txt"},
	}

	expected := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt"}
	allFiles := files.AllFiles()

	assert.Equal(t, expected, allFiles)
}

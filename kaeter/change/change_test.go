package change

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadChangeset(t *testing.T) {
	var tests = []struct {
		name          string
		changeset     string
		expectedError bool
		modifiedFiles int
	}{
		{
			name:          "Invalid changeset fails",
			changeset:     "changeset-invalid.json",
			expectedError: true,
		},
		{
			name:          "Simple changeset",
			changeset:     "changeset-valid.json",
			modifiedFiles: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changesPath := path.Join("test-data", tc.changeset)

			changes, err := LoadChangeset(changesPath)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.modifiedFiles, len(changes.Files.Modified))
			}
		})
	}
}

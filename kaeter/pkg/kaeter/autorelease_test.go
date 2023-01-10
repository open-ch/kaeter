package kaeter

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestAutoRelease(t *testing.T) {
	var tests = []struct {
		name                string
		version             string
		expectError         bool
		expectedYAMLVersion string
	}{
		{
			name:        "Normal version bump",
			version:     "1.2.3",
			expectError: false,
		},
		{
			name:        "Fails to release existing version",
			version:     "0.0.0",
			expectError: true,
		},
		// TODO add a release which doesn't have a changelog entry and fails
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := createMockKaeterRepo(t, emptyMakefileContent, "unit test module init", emptyVersionsYAML)
			defer os.RemoveAll(testFolder)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)
			config := &AutoReleaseConfig{
				Logger:         log.New(),
				ModulePath:     testFolder,
				ReleaseVersion: tc.version,
				RepositoryRef:  "master",
				RepositoryRoot: testFolder,
			}

			err := AutoRelease(config)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				verstionsYaml, err := os.ReadFile(filepath.Join(testFolder, "versions.yaml"))
				assert.NoError(t, err)
				assert.Contains(t, string(verstionsYaml), tc.version)
				assert.Contains(t, string(verstionsYaml), AutoReleaseHash)
			}
		})
	}
}

package actions

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/pkg/mocks"
)

func TestAutoRelease(t *testing.T) {
	var tests = []struct {
		changelogContent    string
		expectedYAMLVersion string
		expectError         bool
		name                string
		skipChangelog       bool
		skipReadme          bool
		version             string
	}{
		{
			name:             "Normal version bump",
			changelogContent: "## 1.4.2 - 25.07.2004 bot",
			version:          "1.4.2",
		},
		{
			name:        "Fails to release existing version",
			version:     "0.0.0",
			expectError: true,
		},
		{
			name:        "Fails when README missing",
			version:     "1.0.0",
			expectError: true,
			skipReadme:  true,
		},
		{
			name:          "Fails when CHANGELOG missing",
			version:       "1.0.0",
			expectError:   true,
			skipChangelog: true,
		},
		{
			name:        "Fails when CHANGELOG doesn't include version",
			version:     "1.0.0",
			expectError: true,
		},
		{
			name:             "Fails when CHANGELOG includes wrong version",
			version:          "2.0.0",
			changelogContent: "## 1.4.2 - 25.07.2004 bot",
			expectError:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateMockKaeterRepo(t, mocks.EmptyMakefileContent, "unit test module init", mocks.EmptyVersionsYAML)
			if !tc.skipReadme {
				mocks.CreateMockFile(t, testFolder, "README.md", "")
			}
			if !tc.skipChangelog {
				mocks.CreateMockFile(t, testFolder, "CHANGELOG.md", tc.changelogContent)
			}
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
				verstionsYaml, err := os.ReadFile(filepath.Join(testFolder, "versions.yaml"))
				assert.NoError(t, err)
				assert.Equal(t, string(verstionsYaml), mocks.EmptyVersionsYAML)
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

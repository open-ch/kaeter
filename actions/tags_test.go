package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

func TestAutoReleaseWithTags(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		tags         []string
		expectError  bool
		skipLint     bool
		isExistingAR bool
	}{
		{
			name:     "autorelease with single tag",
			version:  "1.0.0",
			tags:     []string{"production"},
			skipLint: true,
		},
		{
			name:     "autorelease with multiple tags",
			version:  "1.1.0",
			tags:     []string{"production", "stable", "lts"},
			skipLint: true,
		},
		{
			name:     "autorelease without tags",
			version:  "1.2.0",
			tags:     nil,
			skipLint: true,
		},
		{
			name:     "autorelease with empty tags slice",
			version:  "1.3.0",
			tags:     []string{},
			skipLint: true,
		},
		{
			name:         "bump existing autorelease with new tags",
			version:      "1.0.0",
			tags:         []string{"updated", "tag"},
			skipLint:     true,
			isExistingAR: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			versionsYaml := mocks.EmptyVersionsYAML
			if tc.isExistingAR {
				versionsYaml = mocks.PendingAutoreleaseVersionsYAML
			}

			testFolder, _ := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
				Makefile:          mocks.EmptyMakefileContent,
				VersionsYAML:      versionsYaml,
				READMECreateEmpty: true,
				CHANGELOG:         "## " + tc.version + " - 25.07.2004 bot",
			})

			config := &AutoReleaseConfig{
				ModulePath:     testFolder,
				ReleaseVersion: tc.version,
				Tags:           &tc.tags,
				RepositoryRef:  "main",
				RepositoryRoot: testFolder,
				SkipLint:       tc.skipLint,
			}

			err := AutoRelease(config)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Read the versions file
			versions, err := modules.ReadFromFile(filepath.Join(testFolder, "versions.yaml"))
			assert.NoError(t, err)

			// Get the latest version
			latestVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1]
			assert.Equal(t, tc.version, latestVersion.Number.String())
			assert.Equal(t, AutoReleaseHash, latestVersion.CommitID)

			// Verify tags
			if len(tc.tags) > 0 {
				assert.Equal(t, tc.tags, latestVersion.Tags)

				// Verify tags are in the file
				versionContent, err := os.ReadFile(filepath.Join(testFolder, "versions.yaml"))
				assert.NoError(t, err)
				for _, tag := range tc.tags {
					assert.Contains(t, string(versionContent), tag)
				}
			} else {
				assert.Nil(t, latestVersion.Tags)
			}
		})
	}
}

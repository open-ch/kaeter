package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter//mocks"
	"github.com/open-ch/kaeter//modules"
)

func TestAutoRelease(t *testing.T) {
	var tests = []struct {
		changelogContent    string
		expectedYAMLVersion string
		expectError         bool
		name                string
		skipLint            bool
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
			name:        "Fails to release with empty version and no hooks",
			version:     "",
			expectError: true,
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
		{
			name:             "Allow skipping changelog check",
			version:          "2.0.0",
			changelogContent: "## 1.4.2 - 25.07.2004 bot",
			skipLint:         true,
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
				ModulePath:     testFolder,
				ReleaseVersion: tc.version,
				RepositoryRef:  "master",
				RepositoryRoot: testFolder,
				SkipLint:       tc.skipLint,
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

func TestGetReleaseVersionFromHooks(t *testing.T) {
	var tests = []struct {
		name                string
		expectVersion       string
		expectError         bool
		exportErrorContains string
		releaseVersion      string
		versions            *modules.Versions
	}{
		{
			name:                "Error with no hook defined",
			expectError:         true,
			exportErrorContains: "version to release is required",
			versions:            &modules.Versions{},
		},
		{
			name:                "Error if hook can't be executed",
			expectError:         true,
			exportErrorContains: "no such file or directory",
			versions: &modules.Versions{Metadata: &modules.Metadata{Annotations: map[string]string{
				"open.ch/kaeter-hook/autorelease-version": "test-data/non-existent-hook.sh",
			}}},
		},
		{
			name:                "Error forward from hook",
			expectError:         true,
			exportErrorContains: "error-message",
			versions: &modules.Versions{Metadata: &modules.Metadata{Annotations: map[string]string{
				"open.ch/kaeter-hook/autorelease-version": "test-data/error-hook.sh",
			}}},
		},
		{
			name:          "Hook that returns static version",
			expectVersion: "1.2.3",
			versions: &modules.Versions{Metadata: &modules.Metadata{Annotations: map[string]string{
				"open.ch/kaeter-hook/autorelease-version": "test-data/static-hook.sh",
			}}},
		},
		{
			name:          "Hook that returns version based on arguments (path and current version)",
			expectVersion: "echo-args . 0.4.2",
			versions: &modules.Versions{
				ReleasedVersions: []*modules.VersionMetadata{
					&modules.VersionMetadata{Number: modules.VersionString{Version: "0.1.0"}},
					&modules.VersionMetadata{Number: modules.VersionString{Version: "0.4.2"}},
				},
				Metadata: &modules.Metadata{Annotations: map[string]string{
					"open.ch/kaeter-hook/autorelease-version": "test-data/echo-args-hook.sh",
				}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := &AutoReleaseConfig{
				ModulePath:     ".",
				ReleaseVersion: tc.releaseVersion,
				RepositoryRef:  "master",
				RepositoryRoot: ".",
				versions:       tc.versions,
			}

			version, err := config.getReleaseVersionFromHooks()

			if tc.expectError {
				assert.ErrorContains(t, err, tc.exportErrorContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectVersion, version)
			}
		})
	}
}

package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

func TestAutoRelease(t *testing.T) {
	var tests = []struct {
		changelogContent          string
		expectedYAMLVersion       string
		expectError               bool
		name                      string
		skipLint                  bool
		skipReadme                bool
		version                   string
		customStartingVersionYAML string
	}{
		{
			name:             "Normal version bump",
			changelogContent: "## 1.4.2 - 25.07.2004 bot",
			version:          "1.4.2",
		},
		{
			name:             "Fails to release with empty version and no hooks",
			version:          "",
			changelogContent: "empty",
			expectError:      true,
		},
		{
			name:             "Fails to release existing version",
			changelogContent: "empty",
			version:          "0.0.0",
			expectError:      true,
		},
		{
			name:             "Fails when README missing",
			version:          "1.0.0",
			changelogContent: "empty",
			expectError:      true,
			skipReadme:       true,
		},
		{
			name:        "Fails when CHANGELOG missing",
			version:     "1.0.0",
			expectError: true,
		},
		{
			name:             "Fails when CHANGELOG doesn't include version",
			version:          "1.0.0",
			changelogContent: "no version :(",
			expectError:      true,
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
		{
			name:                      "Bump existing autorelease",
			changelogContent:          "## 1.0.0 - 25.07.2004 bot",
			version:                   "1.0.0",
			customStartingVersionYAML: mocks.PendingAutoreleaseVersionsYAML,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			versionsYaml := mocks.EmptyVersionsYAML
			if tc.customStartingVersionYAML != "" {
				versionsYaml = tc.customStartingVersionYAML
			}
			testFolder, _ := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
				Makefile:          mocks.EmptyMakefileContent,
				VersionsYAML:      versionsYaml,
				READMECreateEmpty: !tc.skipReadme,
				CHANGELOG:         tc.changelogContent,
			})
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
				"open.ch/kaeter-hook/autorelease-version": "testdata/non-existent-hook.sh",
			}}},
		},
		{
			name:                "Error forward from hook",
			expectError:         true,
			exportErrorContains: "error-message",
			versions: &modules.Versions{Metadata: &modules.Metadata{Annotations: map[string]string{
				"open.ch/kaeter-hook/autorelease-version": "testdata/error-hook.sh",
			}}},
		},
		{
			name:          "Hook that returns static version",
			expectVersion: "1.2.3",
			versions: &modules.Versions{Metadata: &modules.Metadata{Annotations: map[string]string{
				"open.ch/kaeter-hook/autorelease-version": "testdata/static-hook.sh",
			}}},
		},
		{
			name:          "Hook that returns version based on arguments (path and current version)",
			expectVersion: "echo-args . 0.4.2",
			versions: &modules.Versions{
				ReleasedVersions: []*modules.VersionMetadata{
					{Number: modules.VersionString{Version: "0.1.0"}},
					{Number: modules.VersionString{Version: "0.4.2"}},
				},
				Metadata: &modules.Metadata{Annotations: map[string]string{
					"open.ch/kaeter-hook/autorelease-version": "testdata/echo-args-hook.sh",
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

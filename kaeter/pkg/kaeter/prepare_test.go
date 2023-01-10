package kaeter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-ch/go-libs/gitshell"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPrepareRelease(t *testing.T) {
	var tests = []struct {
		name                  string
		manualVersion         string
		expectedCommitVersion string
		expectedYAMLVersion   string
	}{
		{
			name:                  "Defaults bumps the patch number",
			expectedCommitVersion: "ch.open.kaeter:unit-test:0.0.1",
			expectedYAMLVersion:   "0.0.1:",
		},
		{
			name:                  "Manual version bump",
			manualVersion:         "1.2.3",
			expectedCommitVersion: "ch.open.kaeter:unit-test:1.2.3",
			expectedYAMLVersion:   "1.2.3:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := createMockKaeterRepo(t, emptyMakefileContent, "unit test module init", emptyVersionsYAML)
			defer os.RemoveAll(testFolder)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)
			config := &PrepareReleaseConfig{
				BumpMajor:           false,
				BumpMinor:           false,
				ModulePaths:         []string{testFolder},
				RepositoryRef:       "master",
				RepositoryRoot:      testFolder,
				UserProvidedVersion: tc.manualVersion,
				Logger:              log.New(),
			}

			err := PrepareRelease(config)

			assert.NoError(t, err)
			commitMsg, err := gitshell.GitCommitMessageFromHash(testFolder, "HEAD")
			assert.NoError(t, err)
			assert.Contains(t, commitMsg, tc.expectedCommitVersion)
			verstionsYaml, err := os.ReadFile(filepath.Join(testFolder, "versions.yaml"))
			assert.NoError(t, err)
			assert.Contains(t, string(verstionsYaml), tc.expectedYAMLVersion)
		})
	}
}

func TestBumpModule(t *testing.T) {
	var tests = []struct {
		name                  string
		doMinorBump           bool
		doMajorBump           bool
		inputVersion          string
		expectedVersionString string
	}{
		{
			name:                  "Defaults bumps the patch number",
			expectedVersionString: "0.0.1",
		},
		{
			name:                  "Defaults bumps the minor number",
			doMinorBump:           true,
			expectedVersionString: "0.1.0",
		},
		{
			name:                  "Defaults bumps the major number",
			doMajorBump:           true,
			expectedVersionString: "1.0.0",
		},
		{
			name:                  "Defaults bumps the major number",
			inputVersion:          "1.2.3",
			expectedVersionString: "1.2.3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := createMockKaeterRepo(t, emptyMakefileContent, "unit test module init", emptyVersionsYAML)
			defer os.RemoveAll(testFolder)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)
			config := &PrepareReleaseConfig{
				BumpMajor:           tc.doMajorBump,
				BumpMinor:           tc.doMinorBump,
				ModulePaths:         []string{},
				RepositoryRef:       "master",
				RepositoryRoot:      testFolder,
				UserProvidedVersion: tc.inputVersion,
				Logger:              log.New(),
			}
			refTime := time.Unix(42, 0)

			versions, err := config.bumpModule(testFolder, "somegithash", &refTime)

			assert.NoError(t, err)
			releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
			assert.Equal(t, releaseVersion, tc.expectedVersionString)
		})
	}
}

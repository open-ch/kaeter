package actions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"github.com/open-ch/kaeter/kaeter/mocks"
)

func TestPrepareRelease(t *testing.T) {
	var tests = []struct {
		changelogContent      string
		expectedCommitVersion string
		expectedFailure       bool
		expectedYAMLVersion   string
		manualVersion         string
		name                  string
		skipChangelog         bool
		skipLint              bool
		skipReadme            bool
	}{
		{
			name:                  "Defaults bumps the patch number",
			expectedCommitVersion: "ch.open.kaeter:unit-test:0.0.1",
			expectedYAMLVersion:   "0.0.1:",
			changelogContent:      "## 0.0.1 - 25.07.2004 bot",
		},
		{
			name:                  "Manual version bump",
			manualVersion:         "1.2.3",
			expectedCommitVersion: "ch.open.kaeter:unit-test:1.2.3",
			expectedYAMLVersion:   "1.2.3:",
			changelogContent:      "## 1.2.3 - 25.07.2004 bot",
		},
		{
			name:          "Skips validation if set",
			skipReadme:    true,
			skipChangelog: true,
			skipLint:      true,
		},
		{
			name:            "Fails validation without readme",
			skipReadme:      true,
			expectedFailure: true,
		},
		{
			name:            "Fails validation without changelog",
			skipChangelog:   true,
			expectedFailure: true,
		},
		{
			name:            "Fails validation with incomplete changelog",
			expectedFailure: true,
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
			logger, _ := test.NewNullLogger()
			config := &PrepareReleaseConfig{
				BumpType:            kaeter.BumpPatch,
				Logger:              logger,
				ModulePaths:         []string{testFolder},
				RepositoryRef:       "master",
				RepositoryRoot:      testFolder,
				SkipLint:            tc.skipLint,
				UserProvidedVersion: tc.manualVersion,
			}

			err := PrepareRelease(config)

			if tc.expectedFailure {
				assert.Error(t, err)
				return
			}
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
		bumpType              kaeter.SemVerBump
		inputVersion          string
		expectedVersionString string
	}{
		{
			name:                  "Defaults bumps the patch number",
			expectedVersionString: "0.0.1",
		},
		{
			name:                  "Defaults bumps the minor number",
			bumpType:              kaeter.BumpMinor,
			expectedVersionString: "0.1.0",
		},
		{
			name:                  "Defaults bumps the major number",
			bumpType:              kaeter.BumpMajor,
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
			testFolder := mocks.CreateMockKaeterRepo(t, mocks.EmptyMakefileContent, "unit test module init", mocks.EmptyVersionsYAML)
			defer os.RemoveAll(testFolder)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)
			logger, _ := test.NewNullLogger() // Makes the output more silent, ideally we could forward to t.Log for output on failures
			config := &PrepareReleaseConfig{
				BumpType:            tc.bumpType,
				ModulePaths:         []string{},
				RepositoryRef:       "master",
				RepositoryRoot:      testFolder,
				UserProvidedVersion: tc.inputVersion,
				Logger:              logger,
			}
			refTime := time.Unix(42, 0)

			versions, err := config.bumpModule(testFolder, "somegithash", &refTime)

			assert.NoError(t, err)
			releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
			assert.Equal(t, releaseVersion, tc.expectedVersionString)
		})
	}
}

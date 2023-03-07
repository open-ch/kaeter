package kaeter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/pkg/mocks"
)

const dryrunMakefileContent = ".PHONY: build test\nbuild:\n\t@echo building\ntest:\n\t@echo testing"
const dummyMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target"
const errorMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo This target fails with error; exit 1"

func TestRunReleasesSkipModules(t *testing.T) {
	commitMessage := "[release] unittest\nRelease Plan:\n```" +
		`lang=yaml
releases:
- ch.open.kaeter:unit-test:0.1.0
` + "```" // no simple way to use a backtick in a raw string...
	versionsYAML := `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT
  0.1.0: 1970-01-01T00:00:00Z|eeeeee`
	// To allow testing if the makefile ran:
	// - build target creates a file called build
	// - test target creates a file called test
	makefileContent := ".PHONY: build test release\nbuild:\n\ttouch build\ntest:\n\ttouch test\nrelease:\n\ttouch release"

	var tests = []struct {
		name        string
		dryRun      bool
		skipModules []string
		hasError    bool
	}{
		{
			name:        "DryRun runs build & test",
			dryRun:      true,
			skipModules: []string{},
			hasError:    false,
		},
		{
			name:        "if Module is skipped nothing happens",
			dryRun:      true,
			skipModules: []string{"ch.open.kaeter:unit-test"},
			hasError:    false,
		},
		{
			name:        "Full release also runs release",
			dryRun:      false,
			skipModules: []string{},
			hasError:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateMockKaeterRepo(t, makefileContent, commitMessage, versionsYAML)
			defer os.RemoveAll(testFolder)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)

			releaseConfig := &ReleaseConfig{
				RepositoryRoot:  testFolder,
				RepositoryTrunk: "origin/master",
				DryRun:          tc.dryRun,
				SkipCheckout:    true,
				SkipModules:     tc.skipModules,
				Logger:          log.New(),
			}

			err := RunReleases(releaseConfig)

			isModuleSkipped := len(tc.skipModules) == 1
			if tc.hasError {
				assert.Error(t, err, tc.name)
			} else {
				assert.NoError(t, err, tc.name)
				buildFileStat, err := os.Stat(filepath.Join(testFolder, "build"))
				if isModuleSkipped {
					assert.Error(t, err, tc.name)
				} else {
					assert.NoError(t, err, tc.name)
					assert.Equal(t, buildFileStat.IsDir(), false, tc.name)
				}
				testFileStat, err := os.Stat(filepath.Join(testFolder, "test"))
				if isModuleSkipped {
					assert.Error(t, err, tc.name)
				} else {
					assert.NoError(t, err, tc.name)
					assert.Equal(t, testFileStat.IsDir(), false, tc.name)
				}
				releaseFileStat, err := os.Stat(filepath.Join(testFolder, "release"))
				if tc.dryRun || isModuleSkipped {
					assert.Error(t, err, tc.name)
				} else {
					assert.NoError(t, err, tc.name)
					assert.Equal(t, releaseFileStat.IsDir(), false, tc.name)
				}
			}
		})
	}
}

func TestRunReleaseProcess(t *testing.T) {
	testFolder := mocks.CreateTmpFolder(t)
	defer os.RemoveAll(testFolder)
	t.Logf("Temp test folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)", testFolder)
	mocks.CreateMockFile(t, testFolder, "versions.yaml", "")
	mocks.CreateMockFile(t, testFolder, "Makefile", dryrunMakefileContent)
	moduleRelease := &moduleRelease{
		releaseConfig: &ReleaseConfig{
			RepositoryRoot:  testFolder,
			RepositoryTrunk: "origin/master",
			DryRun:          true,
			SkipCheckout:    true, // Without this we would need to create a mock repo with git!
			Logger:          log.New(),
			headHash:        "eeeeeeee",
		},
		releaseTarget: ReleaseTarget{
			ModuleID: "ch.open:unit-test",
			Version:  "1.0.0",
		},
		versionsYAMLPath: filepath.Join(testFolder, "versions.yaml"),
		versionsData: &Versions{
			ID:             "ch.open:unit-test",
			ModuleType:     "Makefile",
			VersioningType: "SemVer",
			ReleasedVersions: []*VersionMetadata{
				&VersionMetadata{
					Number:    &VersionNumber{1, 0, 0},
					Timestamp: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CommitID:  "deadbeef",
				},
			},
		},
	}

	err := runReleaseProcess(moduleRelease)

	assert.NoError(t, err)
}

package kaeter

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const dryrunMakefileContent = ".PHONY: build test\nbuild:\n\t@echo building\ntest:\n\t@echo testing"
const dummyMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target"
const errorMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo This target fails with error; exit 1"

// TODO add tests for RunReleases – this might require creating a mock git repo

func TestRunReleaseProcess(t *testing.T) {
	testFolder := createTmpFolder(t)
	defer os.RemoveAll(testFolder)
	t.Logf("Temp test folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)", testFolder)
	createMockFile(t, testFolder, "versions.yaml", "")
	createMockFile(t, testFolder, "Makefile", dryrunMakefileContent)
	moduleRelease := &moduleRelease{
		releaseConfig: &ReleaseConfig{
			RepositoryRoot: testFolder,
			DryRun:         true,
			SkipCheckout:   true,
			Logger:         log.New(),
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
		headHash: "eeeeeeee",
	}

	err := runReleaseProcess(moduleRelease)

	assert.NoError(t, err)
}

func TestDetectModuleMakefile(t *testing.T) {
	var tests = []struct {
		makefiles        []string
		expectedMakefile string
		hasError         bool
	}{
		{
			makefiles:        []string{"Makefile"},
			expectedMakefile: "Makefile",
			hasError:         false,
		},
		{
			makefiles:        []string{"Makefile", "Makefile.kaeter"},
			expectedMakefile: "Makefile.kaeter",
			hasError:         false,
		},
		{
			makefiles:        []string{"Makefile.kaeter"},
			expectedMakefile: "Makefile.kaeter",
			hasError:         false,
		},
		{
			makefiles:        []string{},
			expectedMakefile: "",
			hasError:         true,
		},
	}

	for _, tc := range tests {
		testFolder := createTmpFolder(t)
		defer os.RemoveAll(testFolder)
		for _, makefileMock := range tc.makefiles {
			createMockFile(t, testFolder, makefileMock, dummyMakefileContent)
		}

		makefile, err := detectModuleMakefile(testFolder)

		if tc.hasError {
			assert.Error(t, err)
			assert.Equal(t, "", makefile)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedMakefile, makefile, "Failed detect expected Makefile")
		}
	}
}

func TestRunMakeTarget(t *testing.T) {
	var tests = []struct {
		name            string
		makefileName    string
		makefileContent string
		hasError        bool
	}{
		{
			name:            "Works with regular makefiles",
			makefileName:    "Makefile",
			makefileContent: dummyMakefileContent,
			hasError:        false,
		},
		{
			name:            "Works with Makefile.kaeter",
			makefileName:    "Makefile.kaeter",
			makefileContent: dummyMakefileContent,
			hasError:        false,
		},
		{
			name:            "Fails when make returns error",
			makefileName:    "Makefile.kaeter",
			makefileContent: errorMakefileContent,
			hasError:        true,
		},
		{
			name:            "Fails with no Makefile",
			makefileName:    "Makefile",
			makefileContent: "",
			hasError:        true,
		},
	}

	for _, tc := range tests {
		testFolder := createTmpFolder(t)
		defer os.RemoveAll(testFolder)
		if tc.makefileContent != "" {
			createMockFile(t, testFolder, tc.makefileName, tc.makefileContent)
		}
		var target ReleaseTarget

		err := runMakeTarget(testFolder, tc.makefileName, "snapshot", target)

		if tc.hasError {
			assert.Error(t, err, tc.name)
		} else {
			assert.NoError(t, err, tc.name)
		}
	}
}

func createTmpFolder(t *testing.T) string {
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	return testFolderPath
}

func createMockFile(t *testing.T, tmpPath string, filename string, content string) {
	err := ioutil.WriteFile(filepath.Join(tmpPath, filename), []byte(content), 0644)
	assert.NoError(t, err)
}

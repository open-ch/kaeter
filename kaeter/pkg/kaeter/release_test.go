package kaeter

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-ch/go-libs/gitshell"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
		testFolder := createMockRepo(t, makefileContent, commitMessage, versionsYAML)
		defer os.RemoveAll(testFolder)
		t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)

		releaseConfig := &ReleaseConfig{
			RepositoryRoot: testFolder,
			DryRun:         tc.dryRun,
			SkipCheckout:   true,
			SkipModules:    tc.skipModules,
			Logger:         log.New(),
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
	}
}

func createMockRepo(t *testing.T, makefileContent string, commitMessage string, versionsYAML string) string {
	testFolder := createTmpFolder(t)

	// Our gitshell library doesn't have init or config, so we run these the hard way
	gitInitCommand := exec.Command("git", "init")
	gitInitCommand.Dir = testFolder
	err := gitInitCommand.Run()
	assert.NoError(t, err)

	createMockFile(t, testFolder, "Makefile", makefileContent)
	createMockFile(t, testFolder, "versions.yaml", versionsYAML)
	gitshell.GitAdd(testFolder, ".")

	// Set local user on the tmp repo, to avoid errors when git commit finds no author
	gitConfigEmailCmd := exec.Command("git", "config", "user.email", "unittest@example.ch")
	gitConfigEmailCmd.Dir = testFolder
	gitConfigEmailCmd.Stdout = os.Stdout
	gitConfigEmailCmd.Stderr = os.Stderr
	err = gitConfigEmailCmd.Run()
	assert.NoError(t, err)

	gitshell.GitCommit(testFolder, commitMessage)

	return testFolder
}

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
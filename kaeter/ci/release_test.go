package ci

import (
	"os"
	"path/filepath"

	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/mocks"
)

func TestReleaseSingleModule(t *testing.T) {
	commitMessage := "init"
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
		name               string
		skipModuleCreation bool
		expectedError      bool
		dryrun             bool
	}{
		{
			name:               "Fails if the module cannot be loaded",
			skipModuleCreation: true,
			expectedError:      true,
		},
		{
			name:   "Dry run builds and tests only",
			dryrun: true,
		},
		{
			name: "Release builds, tests and releases",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := "/some/invalid/path"
			if !tc.skipModuleCreation {
				testFolder = mocks.CreateMockKaeterRepo(t, makefileContent, commitMessage, versionsYAML)
				defer os.RemoveAll(testFolder)
				t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolder)
			}

			rc := &ReleaseConfig{
				DryRun:     tc.dryrun,
				ModulePath: testFolder,
			}

			err := rc.ReleaseSingleModule()

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assertFileExists(t, testFolder, "build")
				assertFileExists(t, testFolder, "test")
				if tc.dryrun {
					assertFileDoesNotExist(t, testFolder, "release")
				} else {
					assertFileExists(t, testFolder, "release")
				}
			}
		})
	}
}

func assertFileExists(t *testing.T, testFolder, filename string) {
	t.Helper()
	fileStat, err := os.Stat(filepath.Join(testFolder, filename))
	assert.NoError(t, err)

	assert.Equal(t, fileStat.Mode().IsRegular(), true)
}

func assertFileDoesNotExist(t *testing.T, testFolder, filename string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(testFolder, filename))
	assert.Error(t, err)
}

package ci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

func TestReleaseSingleModule(t *testing.T) {
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
			name: "Release builds and tests and releases",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := "/some/invalid/path"
			if !tc.skipModuleCreation {
				testFolder, _ = mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
					Makefile:     mocks.TouchMakefileContent,
					VersionsYAML: mocks.EmptyVersionsYAML,
				})
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
				return
			}
			assert.NoError(t, err)
			assert.FileExists(t, filepath.Join(testFolder, "build"))
			assert.FileExists(t, filepath.Join(testFolder, "test"))
			if tc.dryrun {
				assert.NoFileExists(t, filepath.Join(testFolder, "release"))
			} else {
				assert.FileExists(t, filepath.Join(testFolder, "release"))
			}
		})
	}
}

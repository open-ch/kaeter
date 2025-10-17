package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

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
			testFolder, _ := mocks.CreateKaeterRepo(t, &mocks.KaeterModuleConfig{
				OverrideCommitMessage: commitMessage,
				Makefile:              mocks.TouchMakefileContent,
				VersionsYAML:          versionsYAML,
			})

			releaseConfig := &ReleaseConfig{
				RepositoryRoot:  testFolder,
				RepositoryTrunk: "origin/main",
				DryRun:          tc.dryRun,
				SkipCheckout:    true,
				SkipModules:     tc.skipModules,
			}

			err := RunReleases(releaseConfig)

			isModuleSkipped := len(tc.skipModules) == 1
			if tc.hasError {
				assert.Error(t, err, tc.name)
				return
			}
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
		})
	}
}

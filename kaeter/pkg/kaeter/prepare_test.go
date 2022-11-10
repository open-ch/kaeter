package kaeter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-ch/go-libs/gitshell"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const emptyMakefileContent = ".PHONY: build test snapshot release"
const emptyVersionsYAML = `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`

func TestPrepareRelease(t *testing.T) {

	var tests = []struct {
		name                  string
		expectedCommitVersion string
		expectedYAMLVersion   string
	}{
		{
			name:                  "Defaults bumps the patch number",
			expectedCommitVersion: "ch.open.kaeter:unit-test:0.0.1",
			expectedYAMLVersion:   "0.0.1:",
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
				UserProvidedVersion: "",
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

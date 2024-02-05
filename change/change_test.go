package change

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

func TestCheck(t *testing.T) {
	var tests = []struct {
		name          string
		expectedInfo  *Information
		expectedError bool
	}{
		{
			name: "Empty changeset",
			expectedInfo: &Information{
				Files: Files{
					Added:    []string{},
					Removed:  []string{},
					Modified: []string{},
				},
				Commit: CommitMsg{
					ReleasePlan: &actions.ReleasePlan{
						Releases: []actions.ReleaseTarget{},
					},
				},
				Kaeter: KaeterChange{
					Modules: map[string]modules.KaeterModule{},
				},
				Helm: HelmChange{
					Charts: []string{},
				},
				PullRequest: &PullRequest{
					ReleasePlan: &actions.ReleasePlan{
						Releases: []actions.ReleaseTarget{},
					},
				},
			},
			expectedError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repoPath := mocks.CreateMockRepo(t)
			defer os.RemoveAll(repoPath)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", repoPath)
			firstCommit := mocks.CommitFileAndGetHash(t, repoPath, "README.md", "# Test Repo", "initial commit")

			detector := &Detector{
				RootPath:       repoPath,
				PreviousCommit: firstCommit,
				CurrentCommit:  firstCommit,
				KaeterModules:  []modules.KaeterModule{}, // TODO run kaeter module detection to build this?
				PullRequest:    &PullRequest{},
			}
			info, err := detector.Check()

			if tc.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedInfo, info)
		})
	}
}

func TestLoadChangeset(t *testing.T) {
	var tests = []struct {
		name          string
		changeset     string
		expectedError bool
		modifiedFiles int
	}{
		{
			name:          "Invalid changeset fails",
			changeset:     "changeset-invalid.json",
			expectedError: true,
		},
		{
			name:          "Simple changeset",
			changeset:     "changeset-valid.json",
			modifiedFiles: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changesPath := path.Join("testdata", tc.changeset)

			changes, err := LoadChangeset(changesPath)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.modifiedFiles, len(changes.Files.Modified))
			}
		})
	}
}

package change

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

var emptyReleasePlan = &actions.ReleasePlan{Releases: []actions.ReleaseTarget{}}

func TestCheck(t *testing.T) {
	var module1 = modules.KaeterModule{
		ModuleID:   "ch.open.kaeter:unit-test",
		ModulePath: "module1",
		ModuleType: "Makefile",
	}
	var module2 = modules.KaeterModule{
		ModuleID:   "ch.open.kaeter:unit-testing",
		ModulePath: "module2",
		ModuleType: "Makefile",
	}
	var tests = []struct {
		name          string
		commitIndexes []int
		expectedInfo  *Information
		expectedError bool
	}{
		{
			name:          "Empty changeset",
			commitIndexes: []int{0, 0},
			expectedInfo: &Information{
				Files: Files{
					Added:    []string{},
					Removed:  []string{},
					Modified: []string{},
				},
				Commit:      CommitMsg{ReleasePlan: emptyReleasePlan},
				Kaeter:      KaeterChange{Modules: map[string]modules.KaeterModule{}},
				Helm:        HelmChange{Charts: []string{}},
				PullRequest: &PullRequest{ReleasePlan: emptyReleasePlan},
			},
		},
		{
			name:          "Small single commit changeset",
			commitIndexes: []int{0, 1},
			expectedInfo: &Information{
				Files: Files{
					Added: []string{
						"module1/Makefile",
						"module1/versions.yaml",
					},
					Removed:  []string{},
					Modified: []string{},
				},
				Commit: CommitMsg{ReleasePlan: emptyReleasePlan},
				Kaeter: KaeterChange{Modules: map[string]modules.KaeterModule{
					module1.ModuleID: module1,
				}},
				Helm:        HelmChange{Charts: []string{}},
				PullRequest: &PullRequest{ReleasePlan: emptyReleasePlan},
			},
		},
		{
			name:          "Multi commit changeset",
			commitIndexes: []int{0, 2},
			expectedInfo: &Information{
				Files: Files{
					Added: []string{
						"module1/Makefile",
						"module1/versions.yaml",
						"module2/Makefile",
						"module2/versions.yaml",
					},
					Removed:  []string{},
					Modified: []string{},
				},
				Commit: CommitMsg{ReleasePlan: emptyReleasePlan},
				Kaeter: KaeterChange{Modules: map[string]modules.KaeterModule{
					module1.ModuleID: module1,
					module2.ModuleID: module2,
				}},
				Helm:        HelmChange{Charts: []string{}},
				PullRequest: &PullRequest{ReleasePlan: emptyReleasePlan},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repoPath, firstCommit := mocks.CreateMockRepo(t)
			_, secondCommit := mocks.CreateKaeterModule(t, repoPath, &mocks.KaeterModuleConfig{
				Path:         "module1",
				Makefile:     mocks.EmptyMakefileContent,
				VersionsYAML: mocks.EmptyVersionsYAML,
			})
			_, thirdCommit := mocks.CreateKaeterModule(t, repoPath, &mocks.KaeterModuleConfig{
				Path:         "module2",
				Makefile:     mocks.EmptyMakefileContent,
				VersionsYAML: mocks.EmptyVersionsAlternateYAML,
			})
			commits := []string{firstCommit, secondCommit, thirdCommit}

			kaeterModules, err := modules.GetKaeterModules(repoPath)
			assert.NoError(t, err)
			detector := &Detector{
				RootPath:       repoPath,
				PreviousCommit: commits[tc.commitIndexes[0]],
				CurrentCommit:  commits[tc.commitIndexes[1]],
				KaeterModules:  kaeterModules,
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

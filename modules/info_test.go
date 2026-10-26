package modules

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-ch/kaeter/mocks"
)

func TestGetNeedsReleaseInfoIn(t *testing.T) {
	var tests = []struct {
		name           string
		mockModule     *mocks.KaeterModuleConfig
		expectedModule *Versions
		expectedID     string
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			expectedID: "ch.open.kaeter:unit-test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateKaeterRepo(t, tc.mockModule)

			modulesChan, err := GetNeedsReleaseInfoIn(testFolder, NeedsReleaseOptions{})

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				modules := []*NeedsReleaseInfo{}
				for needsReleaseInfo := range modulesChan {
					modules = append(modules, &needsReleaseInfo)
				}
				assert.Equal(t, 1, len(modules))
				assert.Equal(t, tc.expectedID, modules[0].ModuleID)
			}
		})
	}
}

func TestLoadModule(t *testing.T) {
	var tests = []struct {
		name           string
		mockModule     *mocks.KaeterModuleConfig
		expectedModule *Versions
		expectedID     string
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			expectedID: "ch.open.kaeter:unit-test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateKaeterRepo(t, tc.mockModule)

			module, err := loadModule(filepath.Join(testFolder, tc.mockModule.Path))

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, module.ID)
			}
		})
	}
}

func TestGetModuleNeedsReleaseInfo(t *testing.T) {
	abitraryReleaseDate := time.Date(2025, time.January, 23, 12, 15, 38, 482146000, time.Local)
	var tests = []struct {
		name                          string
		mockModule                    *mocks.KaeterModuleConfig
		releaseInitialCommit          bool
		addCommits                    bool
		addReleaseOnlyCommit          bool
		includeCommits                bool
		addDependencyDocsCommit       bool
		dependencyIgnore              []string
		expectedCommitSubjects        []string
		expectedModule                *Versions
		expectedID                    string
		expectedPath                  string
		expectedLastRelease           *time.Time
		expectedCommitCount           int
		expectedDependencyCommitCount int
		expectedError                 bool
	}{
		{
			name: "Expect empty module to have no unreleased commits",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           nil,
			expectedCommitCount:           0,
			expectedDependencyCommitCount: 0,
		},
		{
			name: "Expect release to be detected as latest",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			releaseInitialCommit:          true,
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           0,
			expectedDependencyCommitCount: 0,
		},
		{
			name: "Expect changes after release detected (no-dependencies)",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			releaseInitialCommit:          true,
			addCommits:                    true,
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           2,
			expectedDependencyCommitCount: 0,
		},
		{
			name: "Expect changes after release detected (no-dependencies)",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsGoWorkDepYAML,
			},
			releaseInitialCommit:          true,
			addCommits:                    true,
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           2,
			expectedDependencyCommitCount: 1,
		},
		{
			name: "Expect dependency changes detected when the dependency has a leading slash",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsGoWorkLeadingSlashDepYAML,
			},
			releaseInitialCommit:          true,
			addCommits:                    true,
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           2,
			expectedDependencyCommitCount: 1,
		},
		{
			name: "Expect release-only commits to be ignored and commits listed when requested",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			releaseInitialCommit:          true,
			addCommits:                    true,
			addReleaseOnlyCommit:          true,
			includeCommits:                true,
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           2,
			expectedCommitSubjects:        []string{"Add go.mod to module", "Add go.sum to module"},
			expectedDependencyCommitCount: 0,
		},
		{
			name: "Expect Markdown only commits to a dependency to count by default",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: "id: ch.open.kaeter:unit-test\ntype: Makefile\nversioning: SemVer\ndependencies:\n  - go.work\n  - lib\nversions:\n  0.0.0: 1970-01-01T00:00:00Z|INIT",
			},
			releaseInitialCommit:          true,
			addCommits:                    true,
			addDependencyDocsCommit:       true,
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           2,
			expectedDependencyCommitCount: 2,
		},
		{
			name: "Expect commits to a dependency limited to files matching DependencyIgnore not to count",
			mockModule: &mocks.KaeterModuleConfig{
				Path:         "testModule",
				VersionsYAML: "id: ch.open.kaeter:unit-test\ntype: Makefile\nversioning: SemVer\ndependencies:\n  - go.work\n  - lib\nversions:\n  0.0.0: 1970-01-01T00:00:00Z|INIT",
			},
			releaseInitialCommit:          true,
			addCommits:                    true,
			addDependencyDocsCommit:       true,
			dependencyIgnore:              []string{"**/*.md"},
			expectedID:                    "ch.open.kaeter:unit-test",
			expectedPath:                  "testModule",
			expectedLastRelease:           &abitraryReleaseDate,
			expectedCommitCount:           2,
			expectedDependencyCommitCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, kaeterCommitHash := mocks.CreateKaeterRepo(t, tc.mockModule)
			if tc.addCommits {
				_ = mocks.CommitFileAndGetHash(t, testFolder, "testModule/go.mod", "", "Add go.mod to module")
				_ = mocks.CommitFileAndGetHash(t, testFolder, "testModule/go.sum", "", "Add go.sum to module")
				_ = mocks.CommitFileAndGetHash(t, testFolder, "go.work", "", "Add go.work to project root")
			}
			if tc.addDependencyDocsCommit {
				mocks.CreateMockFolder(t, testFolder, "lib")
				_ = mocks.CommitFileAndGetHash(t, testFolder, "lib/README.md", "# lib", "Document the lib")
			}
			if tc.addReleaseOnlyCommit {
				_ = mocks.CommitFileAndGetHash(t, testFolder, "testModule/CHANGELOG.md", "# CHANGELOG", "Release module")
			}
			// Load the module the way kaeter does, after the commits, so declared dependencies exist and are normalized.
			// TODO can we be less file based and mock the git integration for faster tests?
			moduleInfo, err := readKaeterModuleInfo(filepath.Join(testFolder, tc.mockModule.Path, "versions.yaml"), testFolder)
			require.NoError(t, err)
			if tc.releaseInitialCommit {
				moduleInfo.versions.ReleasedVersions = append(moduleInfo.versions.ReleasedVersions, &VersionMetadata{
					Number:    VersionString{Version: "1.4.2"},
					Timestamp: *tc.expectedLastRelease,
					CommitID:  kaeterCommitHash,
				})
			}

			needsReleaseInfo := getModuleNeedsReleaseInfo(&moduleInfo, NeedsReleaseOptions{
				IncludeCommits:   tc.includeCommits,
				DependencyIgnore: tc.dependencyIgnore,
			})

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, needsReleaseInfo.ModuleID)
				assert.Equal(t, tc.expectedPath, needsReleaseInfo.ModulePath)
				assert.Equal(t, tc.expectedLastRelease, needsReleaseInfo.LatestReleaseTimestamp)
				assert.Equal(t, tc.expectedCommitCount, needsReleaseInfo.UnreleasedCommitCount)
				assert.Equal(t, tc.expectedDependencyCommitCount, needsReleaseInfo.UnreleasedDependencyCommitCount)
				assert.Equal(t, tc.expectedCommitSubjects, subjectsOf(needsReleaseInfo.Commits))
			}
		})
	}
}

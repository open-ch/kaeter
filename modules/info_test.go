package modules

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

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

			modulesChan, err := GetNeedsReleaseInfoIn(testFolder)

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

func TestLoadModuleInfo(t *testing.T) {
	var tests = []struct {
		name          string
		mockModule    *mocks.KaeterModuleConfig
		expectedID    string
		expectedError bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",

			mockModule: &mocks.KaeterModuleConfig{
				Path:         "mockModule",
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			expectedID: "ch.open.kaeter:unit-test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateKaeterRepo(t, tc.mockModule)
			moduleFolder := filepath.Join(testFolder, tc.mockModule.Path)
			versionsYamlPath := filepath.Join(moduleFolder, "versions.yaml")

			module, err := loadModuleInfo(versionsYamlPath)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, moduleFolder, module.moduleAbsolutePath)
				assert.Equal(t, "mockModule", module.moduleRelativePath)
				assert.Equal(t, tc.expectedID, module.versions.ID)
				assert.Equal(t, versionsYamlPath, module.versionsYamlAbsolutePath)
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, kaeterCommitHash := mocks.CreateKaeterRepo(t, tc.mockModule)
			modulePath := filepath.Join(testFolder, tc.mockModule.Path)
			versions, err := loadModule(modulePath) // TODO can we be less file based and mock the git integration for faster tests?
			assert.NoError(t, err)
			moduleInfo := &moduleInfo{
				moduleAbsolutePath:       modulePath,
				moduleRelativePath:       "testModule",
				versions:                 versions,
				versionsYamlAbsolutePath: filepath.Join(modulePath, "versions.yaml"),
			}
			if tc.releaseInitialCommit {
				versions.ReleasedVersions = append(versions.ReleasedVersions, &VersionMetadata{
					Number:    VersionString{Version: "1.4.2"},
					Timestamp: *tc.expectedLastRelease,
					CommitID:  kaeterCommitHash,
				})
			}
			if tc.addCommits {
				_ = mocks.CommitFileAndGetHash(t, testFolder, "testModule/go.mod", "", "Add go.mod to module")
				_ = mocks.CommitFileAndGetHash(t, testFolder, "testModule/go.sum", "", "Add go.sum to module")
				_ = mocks.CommitFileAndGetHash(t, testFolder, "go.work", "", "Add go.work to project root")
			}

			needsReleaseInfo := getModuleNeedsReleaseInfo(moduleInfo)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, needsReleaseInfo.ModuleID)
				assert.Equal(t, tc.expectedPath, needsReleaseInfo.ModulePath)
				assert.Equal(t, tc.expectedLastRelease, needsReleaseInfo.LatestReleaseTimestamp)
				assert.Equal(t, tc.expectedCommitCount, needsReleaseInfo.UnreleasedCommitCount)
				assert.Equal(t, tc.expectedDependencyCommitCount, needsReleaseInfo.UnreleasedDependencyCommitCount)
			}
		})
	}
}

func TestCountUnreleasedCommits(t *testing.T) {
	var tests = []struct {
		name          string
		commitLog     string
		ignorePattern string
		expectedCount int
	}{

		{
			name:          "An empty log has 0 commits",
			commitLog:     ``,
			expectedCount: 0,
		},
		{
			name: "By default count all log lines as commit",
			commitLog: `one
			two
			three`,
			expectedCount: 3,
		},
		{
			name: "trailing blanks are ignored",
			commitLog: `one
			two


			`,
			expectedCount: 2,
		},
		{
			name: "By default count all log lines as commit",
			commitLog: `one
			somethingToIgnore hello
			two
			somethingToIgnore there`,
			ignorePattern: "somethingToIgnore",
			expectedCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("needsrelease.ignorepattern", tc.ignorePattern)

			unreleasedCommitCount := countUnreleasedCommits(tc.commitLog)

			assert.Equal(t, tc.expectedCount, unreleasedCommitCount)
		})
	}
}

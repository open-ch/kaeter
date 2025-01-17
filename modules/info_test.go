package modules

import (
	"os"
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
		versionsYAML   string
		expectedModule *Versions
		expectedID     string
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			versionsYAML: `id: ch.open.tools:kaeter
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedID: "ch.open.tools:kaeter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			viper.Reset()
			viper.Set("repoRoot", testFolder)
			defer os.RemoveAll(testFolder)
			_, _ = mocks.AddSubDirKaeterMock(t, testFolder, "testModule", tc.versionsYAML)

			modules, err := GetNeedsReleaseInfoIn(testFolder)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, modules[0].ModuleID)
			}
		})
	}
}

func TestLoadModule(t *testing.T) {
	var tests = []struct {
		name           string
		versionsYAML   string
		expectedModule *Versions
		expectedID     string
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			versionsYAML: `id: ch.open.tools:kaeter
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedID: "ch.open.tools:kaeter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolder)
			modulePath, _ := mocks.AddSubDirKaeterMock(t, testFolder, "testModule", tc.versionsYAML)

			module, err := loadModule(modulePath)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, module.ID)
			}
		})
	}
}

func TestLoadModulesFoundInPath(t *testing.T) {
	var tests = []struct {
		name           string
		m1VersionsYAML string
		m2VersionsYAML string
		expectedModule *Versions
		expectedm1ID   string
		expectedm2ID   string
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			m1VersionsYAML: `id: ch.open.tools:kaeter
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			m2VersionsYAML: `id: ch.open.tools:test
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedm1ID: "ch.open.tools:test",
			expectedm2ID: "ch.open.tools:kaeter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			viper.Reset()
			viper.Set("repoRoot", testFolder)
			defer os.RemoveAll(testFolder)
			_, _ = mocks.AddSubDirKaeterMock(t, testFolder, "testModule1", tc.m1VersionsYAML)
			_, _ = mocks.AddSubDirKaeterMock(t, testFolder, "testModule2", tc.m2VersionsYAML)

			modules, err := loadModulesFoundInPath(testFolder)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 2, len(modules))
				assert.Equal(t, tc.expectedm1ID, modules[0].versions.ID)
				assert.Equal(t, tc.expectedm2ID, modules[1].versions.ID)
			}
		})
	}
}

func TestGetModuleNeedsReleaseInfo(t *testing.T) {
	var tests = []struct {
		name                 string
		versionsYAML         string
		expectedModule       *Versions
		expectedID           string
		expectedPath         string
		expectedLastRelease  *time.Time
		expectedCommitsCount int
		expectedError        bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			versionsYAML: `id: ch.open.tools:kaeter
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedID:          "ch.open.tools:kaeter",
			expectedPath:        "testModule",
			expectedLastRelease: nil,
			// TODO improve mock to have some commits/releases
			expectedCommitsCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolder)
			modulePath, _ := mocks.AddSubDirKaeterMock(t, testFolder, "testModule", tc.versionsYAML)
			t.Log("testFolder:", testFolder)
			t.Log("modulePath:", modulePath)

			// TODO can we be less file based and mock the git integration for faster tests?
			versions, err := loadModule(modulePath)
			assert.NoError(t, err)

			moduleInfo := &moduleInfo{
				moduleAbsolutePath:       modulePath,
				moduleRelativePath:       "testModule",
				versions:                 versions,
				versionsYamlAbsolutePath: filepath.Join(modulePath, "versions.yaml"),
			}

			needsReleaseInfo := getModuleNeedsReleaseInfo(moduleInfo)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, needsReleaseInfo.ModuleID)
				assert.Equal(t, tc.expectedPath, needsReleaseInfo.ModulePath)
				assert.Equal(t, tc.expectedLastRelease, needsReleaseInfo.LatestReleaseTimestamp)
				assert.Equal(t, tc.expectedCommitsCount, needsReleaseInfo.UnreleasedCommitsCount)
			}
		})
	}
}

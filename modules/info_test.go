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
			// TODO move this to the new mocks with kaeter mock config
			_, _ = mocks.AddSubDirKaeterMock(t, testFolder, "testModule", tc.versionsYAML)

			modulesChan, err := GetNeedsReleaseInfoIn(testFolder)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				modules := []*ModuleNeedsReleaseInfo{}
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

func TestLoadModuleInfo(t *testing.T) {
	var tests = []struct {
		name          string
		VersionsYAML  string
		expectedID    string
		expectedError bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			VersionsYAML: `id: ch.open.tools:kaeter
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
			moduleFolder, _ := mocks.AddSubDirKaeterMock(t, testFolder, "mockModule", tc.VersionsYAML)
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

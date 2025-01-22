package modules

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

// TODO test GetKaeterModules

func TestGetRelativeModulePathFrom(t *testing.T) {
	unrelatedRepoRootPath := "/tmp/some-unrelated-path"
	relativeRepoRootPath := "some-unclear-random-relative-path"
	var tests = []struct {
		name        string
		mockDirPath string
		// if set use that instead of the created mock repo folder as repo root, and don't append mock repo folder to versionsYamlPathInRepo:
		mockRepoRoot           *string
		versionsYamlPathInRepo string // the absolute path to repo will be automatically prepended.
		expectedRelativePath   string
		expectedError          bool
	}{
		{
			name:                   "Expects the relative path for a nested folder in the module",
			mockDirPath:            "some/nested/module",
			versionsYamlPathInRepo: "some/nested/module/versions.yaml",
			expectedRelativePath:   "some/nested/module",
		},
		{
			name:                   "Works based on paths even if the sepecific path doesn't exist",
			mockDirPath:            "some/nested/module",
			versionsYamlPathInRepo: "some/other/module/versions.yaml",
			expectedRelativePath:   "some/other/module",
		},
		{
			name:                   "Fails when given relative path for absolute repo",
			mockDirPath:            "some/nested/module",
			mockRepoRoot:           &unrelatedRepoRootPath,
			versionsYamlPathInRepo: "some/relative/path/versions.yaml",
			expectedError:          true,
		},
		{
			name:                   "Adds some path traversal if path is outside repo", // Do we want to fail with error instead or is this allowed?
			mockDirPath:            "some/nested/module",
			mockRepoRoot:           &unrelatedRepoRootPath,
			versionsYamlPathInRepo: "/tmp/some/path/versions.yaml",
			expectedRelativePath:   "../some/path",
		},
		{
			name:                   "Fails when given relative path for relative repo path",
			mockDirPath:            "some/nested/module",
			mockRepoRoot:           &relativeRepoRootPath,
			versionsYamlPathInRepo: "/tmp/some/other/module/versions.yaml",
			expectedError:          true,
		},
		{
			name:                   "Fails for non computable base path",
			mockDirPath:            "some/nested/module",
			mockRepoRoot:           &relativeRepoRootPath,
			versionsYamlPathInRepo: "some/other/module/versions.yaml",
			expectedRelativePath:   "../some/other/module",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolder)
			err := os.MkdirAll(filepath.Join(testFolder, tc.mockDirPath), 0755)
			assert.NoError(t, err)
			versionsYamlPath := tc.versionsYamlPathInRepo
			repoRoot := testFolder
			if tc.mockRepoRoot == nil {
				versionsYamlPath = filepath.Join(testFolder, tc.versionsYamlPathInRepo)
			} else {
				repoRoot = *tc.mockRepoRoot
			}

			relativePath, err := GetRelativeModulePathFrom(versionsYamlPath, repoRoot)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, true, errors.Is(err, ErrModuleRelativePath))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedRelativePath, relativePath)
			}
		})
	}
}

func TestReadKaeterModuleInfo(t *testing.T) {
	var tests = []struct {
		name           string
		versionsYAML   string
		expectedModule KaeterModule
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
			expectedModule: KaeterModule{ModuleID: "ch.open.tools:kaeter", ModulePath: "module", ModuleType: "Makefile"},
		},
		{
			name:           "Expect invalid versions.yaml to fail with error",
			versionsYAML:   `]]clearly__ not yaml [[`,
			expectedError:  true,
			expectedModule: KaeterModule{ModulePath: "module"},
		},
		{
			name: "Expect annotations to be parsed when available",
			versionsYAML: `id: ch.open.osix.pkg:OSAGhello
type: Makefile
versioning: SemVer
metadata:
    annotations:
        open.ch/osix-package: "true"
        open.ch/required-agent-tags: queue=osrp-dev
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedModule: KaeterModule{
				ModuleID:    "ch.open.osix.pkg:OSAGhello",
				ModulePath:  "module",
				ModuleType:  "Makefile",
				Annotations: map[string]string{"open.ch/osix-package": "true", "open.ch/required-agent-tags": "queue=osrp-dev"},
			},
		},
		{
			name: "Detects auto release version",
			versionsYAML: `id: ch.open.tools:unit-test
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
    1.0.0: 1997-08-29T02:14:00Z|AUTORELEASE
`,
			expectedModule: KaeterModule{
				ModuleID:    "ch.open.tools:unit-test",
				ModulePath:  "module",
				ModuleType:  "Makefile",
				AutoRelease: "1.0.0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolder)
			absModulePath, _ := mocks.AddSubDirKaeterMock(t, testFolder, tc.expectedModule.ModulePath, tc.versionsYAML)

			module, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedModule, module)
			}
		})
	}
}

package modules

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-ch/kaeter/mocks"
)

func TestGetKaeterModules(t *testing.T) {
	var tests = []struct {
		name              string
		mockModules       []mocks.KaeterModuleConfig
		searchPathSuffix  string // If not empty join this with the test folder path
		expectedModuleIDs []string
		expectedError     bool
	}{
		{
			name:              "Empty repo has no modules",
			expectedModuleIDs: []string{},
		},
		{
			name: "Detects all modules",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
				{
					Path:         "module2",
					VersionsYAML: mocks.EmptyVersionsAlternateYAML,
				},
			},
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test", "ch.open.kaeter:unit-testing"},
		},
		{
			name: "Fails when invalid module dependencies detected",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path: "module1",
					VersionsYAML: `id: ch.open.kaeter:invalid-deps
type: Makefile
dependencies:
    - not/a/path
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`,
				},
			},
			expectedError: true,
		},

		{
			name: "Detects only modules in the given start path",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "teamA/module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
				{
					Path:         "module2",
					VersionsYAML: mocks.EmptyVersionsAlternateYAML,
				},
			},
			searchPathSuffix:  "teamA",
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			for _, km := range tc.mockModules {
				mocks.CreateKaeterModule(t, testFolder, &km)
			}

			modules, err := GetKaeterModules(filepath.Join(testFolder, tc.searchPathSuffix))

			if tc.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			moduleIDs := make([]string, len(modules))
			for i, mod := range modules {
				moduleIDs[i] = mod.ModuleID
			}
			assert.ElementsMatch(t, tc.expectedModuleIDs, moduleIDs)
		})
	}
}

func TestGetKaeterModules_NormalizesDependencyPaths(t *testing.T) {
	testFolder, _ := mocks.CreateMockRepo(t)
	mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
		Path: "module1",
		VersionsYAML: `id: ch.open.kaeter:leading-slash-deps
type: Makefile
dependencies:
    - /module2
    - ./module2/
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`,
	})
	mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
		Path:         "module2",
		VersionsYAML: mocks.EmptyVersionsAlternateYAML,
	})

	kaeterModules, err := GetKaeterModules(testFolder)
	require.NoError(t, err)

	for _, mod := range kaeterModules {
		if mod.ModuleID == "ch.open.kaeter:leading-slash-deps" {
			assert.Equal(t, []string{"module2", "module2"}, mod.Dependencies)
			// The raw values are kept for writing versions.yaml back on release.
			assert.Equal(t, []string{"/module2", "./module2/"}, mod.GetVersions().Dependencies)
			return
		}
	}
	t.Fatal("module ch.open.kaeter:leading-slash-deps not found")
}

func TestReadKaeterModuleInfo_RejectsInvalidDependencies(t *testing.T) {
	// Each dependency must be reported with the spelling used in versions.yaml.
	for _, dependency := range []string{"", ".", "/", "..", "../", "/not/a/path", "./not/a/path"} {
		t.Run(fmt.Sprintf("dependency '%s'", dependency), func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			absModulePath, _ := mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
				Path:     "module",
				Makefile: mocks.EmptyMakefileContent,
				VersionsYAML: fmt.Sprintf(`id: ch.open.kaeter:invalid-deps
type: Makefile
dependencies:
    - %q
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`, dependency),
			})

			_, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

			require.ErrorIs(t, err, ErrModuleDependencyPath)
			assert.Contains(t, err.Error(), fmt.Sprintf("for '%s'", dependency))
		})
	}
}

func TestNormalizeDependencyPath(t *testing.T) {
	tests := map[string]string{
		"module":        "module",
		"module/":       "module",
		"/module":       "module",
		"//module":      "module",
		"./module":      "module",
		"/module/a.md":  "module/a.md",
		"a/../module":   "module",
		"../module":     "module",
		"/../../module": "module",
		"/":             ".",
		".":             ".",
		"":              ".",
		"..":            ".",
		"../":           ".",
	}
	for dependency, expected := range tests {
		t.Run(dependency, func(t *testing.T) {
			assert.Equal(t, expected, NormalizeDependencyPath(dependency))
		})
	}
}

func TestStreamFoundIn(t *testing.T) {
	var tests = []struct {
		name              string
		mockModules       []mocks.KaeterModuleConfig
		searchPathSuffix  string   // If not empty join this with the test folder path
		ignorePaths       []string // Value of the modules.ignorepaths config
		expectedModuleIDs []string
		expectedError     bool
	}{
		{
			name:              "Empty repo has no modules",
			expectedModuleIDs: []string{},
		},
		{
			name: "Detects all modules",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
				{
					Path:         "module2",
					VersionsYAML: mocks.EmptyVersionsAlternateYAML,
				},
			},
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test", "ch.open.kaeter:unit-testing"},
		},
		{
			name: "Fails when invalid module dependencies detected",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path: "module1",
					VersionsYAML: `id: ch.open.kaeter:invalid-deps
type: Makefile
dependencies:
	- not/a/path
versioning: SemVer
versions:
	0.0.0: 1970-01-01T00:00:00Z|INIT`,
				},
			},
			expectedError: true,
		},
		{
			name: "Detects only modules in the given start path",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path: "teamA/module1",
					VersionsYAML: `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
dependencies:
# relative to repo root:
- module2
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`,
				},
				{
					Path:         "module2",
					VersionsYAML: mocks.EmptyVersionsAlternateYAML,
				},
			},
			searchPathSuffix:  "teamA",
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test"},
		},
		{
			name: "Fails when some modules have duplicate dependencies",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
				{
					Path:         "module2",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
			},
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test"},
			expectedError:     true,
		},
		{
			name: "Skips modules in ignored paths",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
				{
					Path:         "fixtures/module2",
					VersionsYAML: mocks.EmptyVersionsAlternateYAML,
				},
				{
					// Would fail the search with a duplicate ID if it was not skipped
					Path:         "fixtures/nested/module3",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
			},
			ignorePaths:       []string{"fixtures/"},
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test"},
		},
		{
			name: "Ignored paths only match whole directories",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "fixtures-kept/module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
			},
			ignorePaths:       []string{"fixtures"},
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test"},
		},
		{
			name: "Finds nothing when the start path is inside an ignored path",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "fixtures/module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
			},
			searchPathSuffix:  "fixtures/module1",
			ignorePaths:       []string{"fixtures"},
			expectedModuleIDs: []string{},
		},
		{
			name: "Does not let an ignored path cover the whole repository",
			mockModules: []mocks.KaeterModuleConfig{
				{
					Path:         "module1",
					VersionsYAML: mocks.EmptyVersionsYAML,
				},
			},
			ignorePaths:       []string{".", "", "/", "../"},
			expectedModuleIDs: []string{"ch.open.kaeter:unit-test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder, _ := mocks.CreateMockRepo(t)
			for _, km := range tc.mockModules {
				mocks.CreateKaeterModule(t, testFolder, &km)
			}
			if tc.ignorePaths != nil {
				viper.Set(ignorePathsConfigKey, tc.ignorePaths)
				t.Cleanup(func() { viper.Set(ignorePathsConfigKey, nil) })
			}

			resultsChan := StreamFoundIn(filepath.Join(testFolder, tc.searchPathSuffix))
			var moduleIDs []string
			var errs error
			for r := range resultsChan {
				if r.Err != nil {
					errs = errors.Join(errs, r.Err)
				} else {
					moduleIDs = append(moduleIDs, r.Module.ModuleID)
				}
			}

			if tc.expectedError {
				assert.Error(t, errs)
			} else {
				assert.NoError(t, errs)
			}

			assert.ElementsMatch(t, tc.expectedModuleIDs, moduleIDs)
		})
	}
}

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
			mocks.CreateMockFolder(t, testFolder, tc.mockDirPath)
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
			absModulePath, _ := mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
				Path:         tc.expectedModule.ModulePath,
				Makefile:     mocks.EmptyMakefileContent,
				VersionsYAML: tc.versionsYAML,
			})

			module, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.EqualExportedValues(t, tc.expectedModule, module)
			}
		})
	}
}

func TestParseAutorelease_WithTags(t *testing.T) {
	testFolder, _ := mocks.CreateMockRepo(t)

	// Create module with autorelease entry that has tags
	versionsYAML := `id: ch.open.kaeter:test-module
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT
  0.1.0: 2024-01-01T10:00:00Z|AUTORELEASE|unit,integration`

	absModulePath, _ := mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
		Path:         "test-module",
		Makefile:     mocks.EmptyMakefileContent,
		VersionsYAML: versionsYAML,
	})

	module, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

	require.NoError(t, err)
	require.NotNil(t, module)
	assert.Equal(t, "0.1.0", module.AutoRelease, "AutoRelease should be 0.1.0")
	assert.Equal(t, []string{"unit", "integration"}, module.Tags, "Tags should contain unit and integration")
}

func TestParseAutorelease_WithoutTags(t *testing.T) {
	testFolder, _ := mocks.CreateMockRepo(t)

	// Create module with autorelease entry without tags
	versionsYAML := `id: ch.open.kaeter:test-module
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT
  0.1.0: 2024-01-01T10:00:00Z|AUTORELEASE`

	absModulePath, _ := mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
		Path:         "test-module",
		Makefile:     mocks.EmptyMakefileContent,
		VersionsYAML: versionsYAML,
	})

	module, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

	require.NoError(t, err)
	require.NotNil(t, module)
	assert.Equal(t, "0.1.0", module.AutoRelease, "AutoRelease should be 0.1.0")
	assert.Empty(t, module.Tags, "Tags should be empty")
}

func TestParseAutorelease_NoAutorelease(t *testing.T) {
	testFolder, _ := mocks.CreateMockRepo(t)

	// Create module without autorelease entry
	versionsYAML := `id: ch.open.kaeter:test-module
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT
  0.1.0: 2024-01-01T10:00:00Z|abc123def456|unit,integration`

	absModulePath, _ := mocks.CreateKaeterModule(t, testFolder, &mocks.KaeterModuleConfig{
		Path:         "test-module",
		Makefile:     mocks.EmptyMakefileContent,
		VersionsYAML: versionsYAML,
	})

	module, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

	require.NoError(t, err)
	require.NotNil(t, module)
	assert.Empty(t, module.AutoRelease, "AutoRelease should be empty when no autorelease")
	assert.Empty(t, module.Tags, "Tags should be empty when no autorelease")
}

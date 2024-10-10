package lint

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/open-ch/kaeter/mocks"

	"github.com/stretchr/testify/assert"
)

const (
	testDataFolder                  = "testdata"
	existingFile                    = "CHANGELOG"
	existingFolder                  = "testdata" // todo remove
	nonExistingFileInExistingFolder = "random"
	nonExistingFolder               = "any"
)

const versionsYamlMinimal = "id: ch.open.tools:kaeter-police-test"
const versionsYamlWithReleases = `
id: ch.open.tools:kaeter-police-tests
type: Makefile
versioning: SemVer
versions:
    1.0.0: 1970-01-01T00:00:00Z|hash
    1.1.0: 1970-02-01T00:00:00Z|hash
`
const versionsYamlWithVersionDashReleaseReleases = `
id: ch.open.tools:kaeter-police-tests
type: Makefile
versioning: SemVer
versions:
    1.0.1-1: 1970-01-01T00:00:00Z|hash
    1.1.0-1: 1970-02-01T00:00:00Z|hash
`
const versionsYamlAnyStringVer = `
id: ch.open.tools:kaeter-police-tests
type: Makefile
versioning: SemVer
versions:
    v2.8: 1970-01-01T00:00:00Z|hash
    v2.9: 1970-02-01T00:00:00Z|hash
`
const changelogMDWithReleases = `# Changelog
## 1.0.0 - 02.06.2020
 - Initial version
## 1.1.0 - 02.07.2020
 - Minor version
`
const changelogCHANGESWithReleases = `v2.8  17.12.2020 jmj
- something

v2.9  24.06.2021 jmj,pfi
- something more
- something else
`

const specFileName = "something-something.spec"
const specChangelogWithReleases = `Name: testing-spec
Version: 1.1.0
%changelog
* Fri Aug 11 2042 author - 1.1.0-1
- FIX: Fixes the output to always be 42
* Fri Aug 1 2042 author - 1.0.1-1
- TRIVIAL: Initial version release
`

type mockModule struct {
	versions      string
	readme        string
	changelog     string
	changelogName string
}

func TestCheckModulesStartingFrom(t *testing.T) {
	tests := []struct {
		name string
		// createRepo takes T in case you need to pass it an error,
		// we return 2 paths:
		// - the base path for clean up post test case
		// - the path we want to test, allowning each test case to define it's own test path
		createRepo func(t *testing.T) (string, string)
		hasError   bool
	}{
		{
			name: "Fails if path isn't within a git repo",
			createRepo: func(t *testing.T) (string, string) {
				repoPath := mocks.CreateTmpFolder(t)
				return repoPath, repoPath
			},
			hasError: true,
		},
		{
			name: "No error for path with no modules (empty repo)",
			createRepo: func(t *testing.T) (string, string) {
				repoPath, _ := mocks.CreateMockRepo(t)
				testDir := path.Join(repoPath, "test")
				err := os.Mkdir(testDir, 0755)
				assert.NoError(t, err)
				return repoPath, testDir
			},
		},
		{
			name: "Fails if a kaeter module without changelog is found",
			createRepo: func(t *testing.T) (string, string) {
				repoPath := mocks.CreateMockKaeterRepo(t, "", "init", "")
				return repoPath, repoPath
			},
			hasError: true,
		},
		{
			name: "Finds on invalid module (no changelog) even if given a nested path in repo",
			createRepo: func(t *testing.T) (string, string) {
				repoPath := mocks.CreateMockKaeterRepo(t, "", "init", "")
				testDir := path.Join(repoPath, "test")
				err := os.Mkdir(testDir, 0755)
				assert.NoError(t, err)
				return repoPath, testDir
			},
			hasError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repoPath, testDir := tc.createRepo(t)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(repoPath)` to keep for debugging)\n", repoPath)
			defer os.RemoveAll(repoPath)

			err := CheckModulesStartingFrom(testDir)

			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCheckModuleFromVersionsFile(t *testing.T) {
	tests := []struct {
		name   string
		module mockModule
		valid  bool
	}{
		{
			name:   "pass when all OK with changelog.md",
			module: mockModule{versions: versionsYamlWithReleases, readme: "Test", changelog: changelogMDWithReleases, changelogName: changelogMDFile},
			valid:  true,
		},
		{
			name:   "pass when all OK with CHANGES",
			module: mockModule{versions: versionsYamlAnyStringVer, readme: "Test", changelog: changelogCHANGESWithReleases, changelogName: changelogCHANGESFile},
			valid:  true,
		},
		{
			name:   "pass when all OK with .spec",
			module: mockModule{versions: versionsYamlWithVersionDashReleaseReleases, readme: "Test", changelog: specChangelogWithReleases, changelogName: specFileName},
			valid:  true,
		},
		{
			name:   "fails if readme missing",
			module: mockModule{versions: versionsYamlMinimal, readme: "", changelog: "Changelog", changelogName: changelogMDFile},
			valid:  false,
		},
		{
			name:   "fails if changelog missing",
			module: mockModule{versions: versionsYamlMinimal, readme: "Test", changelog: "", changelogName: changelogMDFile},
			valid:  false,
		},
		{
			name:   "fails if changelog.md incomplete",
			module: mockModule{versions: versionsYamlWithReleases, readme: "Test", changelog: "Changelog", changelogName: changelogMDFile},
			valid:  false,
		},
		{
			name:   "fails if CHANGES incomplete",
			module: mockModule{versions: versionsYamlAnyStringVer, readme: "Test", changelog: "Missing Releases", changelogName: changelogCHANGESFile},
			valid:  false,
		},
		{
			name:   "fails if .spec file changelog incomplete",
			module: mockModule{versions: versionsYamlAnyStringVer, readme: "Test", changelog: "# Incomplete", changelogName: specFileName},
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modulePath := createMockModuleWith(t, tt.module)
			defer os.RemoveAll(modulePath)
			t.Logf("tmp modulePath: %s (comment out the defer os.RemoveAll to keep folder after tests)", modulePath)

			err := CheckModuleFromVersionsFile(path.Join(modulePath, "versions.yaml"))

			if tt.valid {
				assert.NoError(t, err, tt.name)
			} else {
				assert.Error(t, err, tt.name)
			}
		})
	}
}

func TestCheckExistence(t *testing.T) {
	absTestFolder, err := filepath.Abs(testDataFolder)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		testPath string
		testFile string
		hasError bool
	}{
		{
			name:     "works on relative folder existing file",
			testPath: testDataFolder,
			testFile: existingFile,
		},
		{
			name:     "rejects on relative folder missing file",
			testPath: testDataFolder,
			testFile: nonExistingFileInExistingFolder,
			hasError: true,
		},
		{
			name:     "rejects non existent relative folder",
			testPath: nonExistingFolder,
			hasError: true,
		},
		{
			name:     "works on absolute folder existing file",
			testPath: absTestFolder,
			testFile: existingFile,
		},
		{
			name:     "rejects on absolute folder missing file",
			testPath: absTestFolder,
			testFile: nonExistingFileInExistingFolder,
			hasError: true,
		},
		{
			name:     "rejects non existent absolute folder",
			testPath: "/tmp/some/really/unlikely/path12341234",
			hasError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkExistence(tc.testFile, tc.testPath)

			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TODO is this still needed now that we have the mocks module?
func createMockModuleWith(t *testing.T, module mockModule) (modulePath string) {
	modulePath, err := os.MkdirTemp("", "kaeter-police-*")
	assert.NoError(t, err)

	err = os.WriteFile(path.Join(modulePath, "versions.yaml"), []byte(module.versions), 0600)
	assert.NoError(t, err)

	if module.readme != "" {
		err = os.WriteFile(path.Join(modulePath, readmeFile), []byte(module.readme), 0600)
		assert.NoError(t, err)
	}

	if module.changelog != "" && module.changelogName != "" {
		err = os.WriteFile(path.Join(modulePath, module.changelogName), []byte(module.changelog), 0600)
		assert.NoError(t, err)
	}

	return modulePath
}

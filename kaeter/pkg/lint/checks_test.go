package lint

import (
	"fmt"
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	existingFile                    = "CHANGELOG"
	existingFolder                  = "test-data"
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
versioning: AnyStringVer
versions:
    1.0.1-1: 1970-01-01T00:00:00Z|hash
    1.1.0-1: 1970-02-01T00:00:00Z|hash
`
const versionsYamlAnyStringVer = `
id: ch.open.tools:kaeter-police-tests
type: Makefile
versioning: AnyStringVer
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

func TestCheckModulesStartingFromNoModules(t *testing.T) {
	repoPath := createMockRepoFolder(t)
	testPath := path.Join(repoPath, "test")
	defer os.RemoveAll(repoPath)

	err := CheckModulesStartingFrom(testPath)

	assert.NoError(t, err)
}

func TestCheckModulesStartingFromInvalidModules(t *testing.T) {
	repoPath := createMockRepoFolder(t)
	testPath := path.Join(repoPath, "test")
	err := os.WriteFile(path.Join(repoPath, "versions.yaml"), []byte(versionsYamlMinimal), 0655)

	defer os.RemoveAll(repoPath)

	err = CheckModulesStartingFrom(testPath)

	assert.Error(t, err)
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

			err := checkModuleFromVersionsFile(path.Join(modulePath, "versions.yaml"))

			if tt.valid {
				assert.NoError(t, err, tt.name)
			} else {
				assert.Error(t, err, tt.name)
			}
		})
	}
}

func TestCheckExistenceRelative(t *testing.T) {
	// CHANGELOG exists
	err := checkExistence(existingFile, existingFolder)
	assert.NoError(t, err)

	// error due to the non-existence of the file "random" inside the existing folder test-data
	err = checkExistence(nonExistingFileInExistingFolder, existingFolder)
	errMsg := fmt.Sprintf(
		"Error in getting FileInfo about '%s': %s",
		nonExistingFileInExistingFolder,
		fmt.Sprintf("stat %s: no such file or directory", filepath.Join(existingFolder, nonExistingFileInExistingFolder)),
	)
	assert.EqualError(t, err, errMsg)

	// error due to the non-existence of the folder "any"
	err = checkExistence(existingFile, nonExistingFolder)
	errMsg = fmt.Sprintf(
		"Error in getting FileInfo about '%s': %s",
		nonExistingFolder,
		fmt.Sprintf("stat %s: no such file or directory", nonExistingFolder),
	)
	assert.EqualError(t, err, errMsg)
}

func TestCheckExistenceAbsolute(t *testing.T) {
	// getting absolute path for test-data
	abs, err := filepath.Abs(existingFolder)
	assert.NoError(t, err)

	// CHANGELOG exists
	err = checkExistence(existingFile, abs)
	assert.NoError(t, err)

	// error due to the non-existence of the file "random" inside the existing folder test-data
	err = checkExistence(nonExistingFileInExistingFolder, abs)
	errMsg := fmt.Sprintf(
		"Error in getting FileInfo about '%s': %s",
		nonExistingFileInExistingFolder,
		fmt.Sprintf("stat %s: no such file or directory", filepath.Join(abs, nonExistingFileInExistingFolder)),
	)
	assert.EqualError(t, err, errMsg)

	// getting absolute path for a non-existing folder
	abs, err = filepath.Abs(nonExistingFolder)
	assert.NoError(t, err)

	// error due to the non-existence of the folder "any"
	err = checkExistence(existingFile, abs)
	errMsg = fmt.Sprintf("Error in getting FileInfo about '%s': %s", abs, fmt.Sprintf("stat %s: no such file or directory", abs))
	assert.EqualError(t, err, errMsg)
}

func TestCheckChangelog(t *testing.T) {
	testDataPath, err := filepath.Abs(existingFolder)
	assert.NoError(t, err)
	versionsFilePath := path.Join(testDataPath, "dummy-versions-valid")
	versions, err := kaeter.ReadFromFile(versionsFilePath)
	assert.NoError(t, err)
	changelogFilePath := path.Join(testDataPath, "dummy-changelog-SemVer")

	err = checkChangelog(changelogFilePath, versions)

	assert.NoError(t, err)
}

func createMockRepoFolder(t *testing.T) (repoPath string) {
	repoPath, err := os.MkdirTemp("", "kaeter-police-*")
	assert.NoError(t, err)

	err = os.Mkdir(path.Join(repoPath, ".git"), 0755)
	assert.NoError(t, err)

	err = os.Mkdir(path.Join(repoPath, "test"), 0755)
	assert.NoError(t, err)

	return repoPath
}

func createMockModuleWith(t *testing.T, module mockModule) (modulePath string) {
	modulePath, err := os.MkdirTemp("", "kaeter-police-*")
	assert.NoError(t, err)

	err = os.WriteFile(path.Join(modulePath, "versions.yaml"), []byte(module.versions), 0655)
	assert.NoError(t, err)

	if module.readme != "" {
		err = os.WriteFile(path.Join(modulePath, readmeFile), []byte(module.readme), 0655)
		assert.NoError(t, err)
	}

	if module.changelog != "" && module.changelogName != "" {
		err = os.WriteFile(path.Join(modulePath, module.changelogName), []byte(module.changelog), 0655)
		assert.NoError(t, err)
	}

	return modulePath
}

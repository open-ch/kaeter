package mocks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// EmptyMakefileContent is the content of the minimal Makefile, used for testing
const EmptyMakefileContent = ".PHONY: build test snapshot release"

// TouchMakefileContent is a makefile useful for testing releases and such each target will simply
// touch a file matching the target name allowing easily checking from the outside if targets were
// called or not.
const TouchMakefileContent = ".PHONY: build test release\nbuild:\n\ttouch build\ntest:\n\ttouch test\nrelease:\n\ttouch release"

// EmptyVersionsYAML is the content of a minimal kaeter versions file, used for testing
const EmptyVersionsYAML = `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`

// EmptyVersionsGoWorkDepYAML is the content of a minimal kaeter versions file with
// a declared dependency on go.work, useful for dependency related tests
const EmptyVersionsGoWorkDepYAML = `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
dependencies:
  - go.work
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`

// EmptyVersionsAlternateYAML is a minimal kaeter versions file content with a differen module id
const EmptyVersionsAlternateYAML = `id: ch.open.kaeter:unit-testing
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT`

// PendingAutoreleaseVersionsYAML is the content of a minimal kaeter versions file with a
// 1.0.0 AUTORELEASE pending.
const PendingAutoreleaseVersionsYAML = `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
versions:
  0.0.0: 1970-01-01T00:00:00Z|INIT
  1.0.0: 1970-01-01T00:00:00Z|AUTORELEASE`

// KaeterModuleConfig configures how to create a kaeter mock
type KaeterModuleConfig struct {
	Path                    string // Sub folder of module relative to root folder, if empty module is created at root
	CHANGELOG               string
	CHANGELOGCreateEmpty    bool   // If true create empty file, otherwise when VersionsYAML is "" file wont be created
	CHANGELOGName           string // Handy to create a CHANGELOG.md or CHANGE or something.spec changelog (default CHANGELOG.md)
	Makefile                string
	MakefileCreateEmpty     bool
	MakefileDotKaeter       bool   // Create a Makefile.kaeter or plain Makefile
	OverrideCommitMessage   string // If not empty use this as commit message when adding files
	README                  string
	READMECreateEmpty       bool
	VersionsYAML            string
	VersionsYAMLCreateEmpty bool
}

// CreateKaeterRepo is a test helper to create a mock kaeter module in a tmp fodler
// it returns the path to the tmp folder. Caller is responsible for deleting it.
func CreateKaeterRepo(t *testing.T, module *KaeterModuleConfig) (repoMockPath, keaterModuleCommitHash string) {
	t.Helper()
	testFolder, _ := CreateMockRepo(t)
	_, kaeterCommitHash := CreateKaeterModule(t, testFolder, module)
	return testFolder, kaeterCommitHash
}

// CreateKaeterModule is a test helper to initialize a mock kaeter module in an existing folder
// a KaeterModuleConfig config is used to decide how and which files to initialize
func CreateKaeterModule(t *testing.T, testFolder string, module *KaeterModuleConfig) (moduleFolder, commitHash string) {
	t.Helper()

	modulePath := CreateMockFolder(t, testFolder, module.Path)
	commitMessage := fmt.Sprintf("Add module %s", modulePath)
	makefileName := "Makefile"
	changelogFilename := "CHANGELOG.md"
	if module.MakefileDotKaeter {
		makefileName = "Makefile.kaeter"
	}
	if module.CHANGELOGName != "" {
		changelogFilename = module.CHANGELOGName
	}
	if module.OverrideCommitMessage != "" {
		commitMessage = module.OverrideCommitMessage
	}

	if module.VersionsYAMLCreateEmpty || module.VersionsYAML != "" {
		CreateMockFile(t, modulePath, "versions.yaml", module.VersionsYAML)
	}
	if module.READMECreateEmpty || module.README != "" {
		CreateMockFile(t, modulePath, "README.md", module.README)
	}
	if module.CHANGELOGCreateEmpty || module.CHANGELOG != "" {
		CreateMockFile(t, modulePath, changelogFilename, module.CHANGELOG)
	}
	if module.MakefileCreateEmpty || module.Makefile != "" {
		CreateMockFile(t, modulePath, makefileName, module.Makefile)
	}
	execGitCommand(t, modulePath, "add", ".")
	execGitCommand(t, modulePath, "commit", "-m", commitMessage)
	commitHash = execGitCommand(t, modulePath, "rev-parse", "--verify", "HEAD")

	return modulePath, commitHash
}

// CreateMockRepo initializes a mock git repository in a tmp folder
// Note that it will also reset viper set the repoRoot key to the test folder as a convenience
func CreateMockRepo(t *testing.T) (folder, commitHash string) {
	t.Helper()
	testFolder := CreateTmpFolder(t)

	// Our git wrapper doesn't have init or config so we do it inline here
	execGitCommand(t, testFolder, "init")

	// Set local user on the tmp repo, to avoid errors when git commit finds no author
	execGitCommand(t, testFolder, "config", "user.email", "unittest@example.ch")
	execGitCommand(t, testFolder, "config", "user.name", "Unit Test")

	// Note:
	// To support older versions git that don't support renaming the branch,
	// the default new repo branch is master.
	// It's possible it randomly changes to main once we update one day and this
	// tests starts failing.
	// However atempts to change to a deterministic branch (i.e. test)
	// consistently failed to run on CI
	// git init --initial-branch test -> not supported on older git versions
	// git branch -M test -> fails to rename the branch

	CreateMockFile(t, testFolder, "README.md", "# Test repo")
	execGitCommand(t, testFolder, "add", ".")
	execGitCommand(t, testFolder, "commit", "-m", "initial commit")
	commitHash = execGitCommand(t, testFolder, "rev-parse", "--verify", "HEAD")

	viper.Reset()
	viper.Set("repoRoot", testFolder)

	return testFolder, commitHash
}

// CommitFileAndGetHash wrapper around git add and git commit, returns the hash of commit.
func CommitFileAndGetHash(t *testing.T, repoPath, filename, fileContent, commitMessage string) string {
	t.Helper()
	CreateMockFile(t, repoPath, filename, fileContent)

	execGitCommand(t, repoPath, "add", ".")
	execGitCommand(t, repoPath, "commit", "-m", commitMessage)
	return execGitCommand(t, repoPath, "rev-parse", "--verify", "HEAD")
}

// SwitchToNewBranch wrapper around git switch -c branchName
func SwitchToNewBranch(t *testing.T, repoPath, branchName string) {
	t.Helper()

	execGitCommand(t, repoPath, "switch", "-c", branchName)
}

// CreateTmpFolder returns path to new temp folder for testing
func CreateTmpFolder(t *testing.T) string {
	t.Helper()
	//nolint:usetesting // We don't use t.TempDir() so we can disable the clean up below when debugging kaeter issues
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	t.Logf("Temp folder: %s\n(disable `t.Cleanup()` in `mocks.go to keep for debugging)\n", testFolderPath)
	t.Cleanup(func() {
		err := os.RemoveAll(testFolderPath)
		assert.NoError(t, err)
	})

	return testFolderPath
}

// CreateMockFile creates file with content in a tmp folder
func CreateMockFile(t *testing.T, tmpPath, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(tmpPath, filename), []byte(content), 0600)
	assert.NoError(t, err)
}

// CreateMockFolder mock folder or folder structure in a given tmp folder
// returns the absolute path of created folder
func CreateMockFolder(t *testing.T, tmpPath, folderPath string) string {
	t.Helper()
	finalPath := filepath.Join(tmpPath, folderPath)
	err := os.MkdirAll(finalPath, 0755)
	assert.NoError(t, err)
	return finalPath
}

// GetEmptyVersionsYaml generates a new empty versions yaml with the given moduleID
// useful to create multiple modules without having duplicate ids.
func GetEmptyVersionsYaml(t *testing.T, moduleID string) string {
	t.Helper()
	return strings.ReplaceAll(EmptyVersionsYAML, "ch.open.kaeter:unit-test", moduleID)
}

func execGitCommand(t *testing.T, repoPath string, additionalArgs ...string) string {
	t.Helper()

	gitCmd := exec.Command("git", additionalArgs...)
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	t.Log(string(output))
	assert.NoError(t, err)

	return strings.TrimSpace(string(output))
}

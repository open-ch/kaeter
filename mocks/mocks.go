package mocks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"testing"

	"github.com/stretchr/testify/assert"
)

// EmptyMakefileContent is the content of the minimal Makefile, used for testing
const EmptyMakefileContent = ".PHONY: build test snapshot release"

// EmptyVersionsYAML is the content of a minimal kaeter versions file, used for testing
const EmptyVersionsYAML = `id: ch.open.kaeter:unit-test
type: Makefile
versioning: SemVer
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

// CreateMockKaeterRepo is a test helper to create a mock kaeter module in a tmp fodler
// it returns the path to the tmp folder. Caller is responsible for deleting it.
// Deprecated: use CreateKaeterRepo instead which offers more flexibility and avoids duplicate spelling of
// mock in signature.
// TODO refactor uses of CreateMockKaeterRepo to CreateKaeterRepo
func CreateMockKaeterRepo(t *testing.T, makefileContent, commitMessage, versionsYAML string) string {
	t.Helper()
	testFolder, _ := CreateMockRepo(t)

	_ = CreateKaeterModule(t, testFolder, &KaeterModuleConfig{
		OverrideCommitMessage: commitMessage,
		Makefile:              makefileContent,
		VersionsYAML:          versionsYAML,
	})

	return testFolder
}

// CreateKaeterRepo is a test helper to create a mock kaeter module in a tmp fodler
// it returns the path to the tmp folder. Caller is responsible for deleting it.
func CreateKaeterRepo(t *testing.T, module *KaeterModuleConfig) string {
	t.Helper()
	testFolder, _ := CreateMockRepo(t)
	_ = CreateKaeterModule(t, testFolder, module)
	return testFolder
}

// AddSubDirKaeterMock is a test helper to create a mock kaeter module in a tmp fodler
// it returns the path to the tmp folder. Caller is responsible for deleting it.
// TODO refactor to use KaeterModuleConfig as argument
func AddSubDirKaeterMock(t *testing.T, testFolder, modulePath, versionsYAML string) (moduleFolder, commitHash string) {
	t.Helper()

	absPath := testFolder
	if modulePath != "." { // Only create sub folders if needed
		absPath = filepath.Join(testFolder, modulePath)
		err := os.Mkdir(absPath, 0755)
		assert.NoError(t, err)
	}

	commitHash = CreateKaeterModule(t, absPath, &KaeterModuleConfig{
		Makefile:     EmptyMakefileContent,
		VersionsYAML: versionsYAML,
	})
	return absPath, commitHash
}

// CreateKaeterModule is a test helper to initialize a mock kaeter module in an existing folder
// a KaeterModuleConfig config is used to decide how and which files to initialize
func CreateKaeterModule(t *testing.T, modulePath string, module *KaeterModuleConfig) (commitHash string) {
	t.Helper()

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

	return commitHash
}

// CreateMockRepo initializes a mock git repository in a tmp folder
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

func execGitCommand(t *testing.T, repoPath string, additionalArgs ...string) string {
	t.Helper()

	gitCmd := exec.Command("git", additionalArgs...)
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	t.Log(string(output))
	assert.NoError(t, err)

	return strings.TrimSpace(string(output))
}

// CreateTmpFolder returns path to new temp folder for testing
func CreateTmpFolder(t *testing.T) string {
	t.Helper()
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	return testFolderPath
}

// CreateMockFile creates file with content in a tmp folder
func CreateMockFile(t *testing.T, tmpPath, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(tmpPath, filename), []byte(content), 0600)
	assert.NoError(t, err)
}

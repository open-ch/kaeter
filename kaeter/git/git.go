package git

import (
	"os/exec"
)

// TODOs
// allow setting repoPath as a global state (singleton)

// Restore simplifies calls to git restore args...
//
//	git.Restore("path/to/unstaged/change")
//	git.Restore("--staged", "path/to/staged/change")
//	git.Restore("--staged", "--worktree", "path/to/some/change")
func Restore(repoPath string, additionalArgs ...string) (string, error) {
	return git(repoPath, "restore", additionalArgs...)
}

// git is a wrapper around exec.Command to simplify the implementation
// of multiple commands and make this file more dry.
func git(repoPath, subCommand string, additionalArgs ...string) (string, error) {
	gitCmd := exec.Command("git", append([]string{subCommand}, additionalArgs...)...)
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	return string(output), err
}

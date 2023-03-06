package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// TODOs
// allow setting repoPath as a global state (singleton)

// Restore simplifies calls to git restore args...
//
//	git.Restore("path/to/unstaged/change")
//	git.Restore("path/to/staged/change", "--staged")
//	git.Restore("path/to/some/change", "--staged", "--worktree")
func Restore(repoPath string, additionalArgs ...string) (string, error) {
	return git(repoPath, "restore", additionalArgs...)
}

// BranchContains is a shortcut to check
//
//	git.BranchContains("path/to/repo", "commit_hash", "branch_pattern")
//	git.BranchContains("path/to/repo", "3ae22a91a12", "main")
func BranchContains(repoPath, hash, pattern string) (string, error) {
	return git(repoPath, "branch", "--all", "--contains", hash, "--list", pattern)
}

// ResolveRevision prints the SHA1 hash given a revision specifier
// see https://git-scm.com/docs/git-rev-parse for more details
func ResolveRevision(repoPath, rev string) (string, error) {
	// --verify gives us a more compact error output
	output, err := git(repoPath, "rev-parse", "--verify", rev)
	if err != nil {
		if notFound, _ := regexp.MatchString("fatal: Needed a single revision", output); notFound {
			return output, fmt.Errorf("error cannot resolve passed commit identifier: %s", rev)
		}
		return output, err
	}

	return strings.TrimSpace(output), nil
}

// git is a wrapper around exec.Command to simplify the implementation
// of multiple commands and make this file more dry.
func git(repoPath, subCommand string, additionalArgs ...string) (string, error) {
	gitCmd := exec.Command("git", append([]string{subCommand}, additionalArgs...)...)
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	return string(output), err
}

package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Add simplifies calls to git add path
//
//	git.Add("path/to/repo", "README.md")
func Add(repoPath, path string) (string, error) {
	return git(repoPath, "add", path)
}

// Commit simplifies calls to git commit -m message
//
//	git.Commit("path/to/repo", "FIX: solves the concurency problem with the turbolift")
func Commit(repoPath, message string) (string, error) {
	return git(repoPath, "commit", "-m", message)
}

// Checkout simplifies calls to git checkout gitRef
//
//	git.Checkout("path/to/repo", "d69928b5a74f70f6000db39d63d84e0aa2aa8ec9")
func Checkout(repoPath, ref string) (string, error) {
	return git(repoPath, "checkout", ref)
}

// ResetHard simplifies calls to git reset --hard gitRef
//
//	git.ResetHard("path/to/repo", "d69928b5a74f70f6000db39d63d84e0aa2aa8ec9")
func ResetHard(repoPath, ref string) (string, error) {
	return git(repoPath, "reset", "--hard", ref)
}

// Restore simplifies calls to git restore args...
//
//	git.Restore("path/to/unstaged/change")
//	git.Restore("path/to/staged/change", "--staged")
//	git.Restore("path/to/some/change", "--staged", "--worktree")
func Restore(repoPath string, additionalArgs ...string) (string, error) {
	return git(repoPath, "restore", additionalArgs...)
}

// Log simplifies calls to git log args...
//
//	git.Log("path/to/unstaged/change")
//	git.Log("path/to/staged/change", "--oneline")
//	git.Log("path/to/some/change", "--oneline", "some/path")
func Log(repoPath string, additionalArgs ...string) (string, error) {
	return git(repoPath, "log", additionalArgs...)
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
		if notFound, _ := regexp.MatchString("fatal: Needed a single revision", output); notFound { //nolint:errcheck
			return output, fmt.Errorf("error cannot resolve passed commit identifier: %s", rev)
		}
		return output, err
	}

	return strings.TrimSpace(output), nil
}

// ShowTopLevel finds the root of a git repo given a path
// see https://git-scm.com/docs/git-rev-parse#Documentation/git-rev-parse.txt---show-toplevel for more details
func ShowTopLevel(repoPath string) (string, error) {
	output, err := git(repoPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return output, err
	}

	return strings.TrimSuffix(output, "\n"), nil
}

// GetCommitMessageFromRef returns the commit message (raw body) from the given commit revision
// or hash.
// see https://git-scm.com/docs/git-log for more details
func GetCommitMessageFromRef(repoPath, rev string) (string, error) {
	return git(repoPath, "log", "-n", "1", "--pretty=format:%B", rev)
}

// git is a wrapper around exec.Command to simplify the implementation
// of multiple commands and make this file more dry.
func git(repoPath, subCommand string, additionalArgs ...string) (string, error) {
	gitCmd := exec.Command("git", append([]string{subCommand}, additionalArgs...)...)
	gitCmd.Dir = repoPath
	output, err := gitCmd.CombinedOutput()
	return string(output), err
}

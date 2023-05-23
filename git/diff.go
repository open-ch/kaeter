package git

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// FileChangeStatus is an enumeration of possible actions perform on files within a commit.
type FileChangeStatus int

var spacesRegex = regexp.MustCompile(`\s+`)

const (
	// Modified signals that the file was modified
	Modified FileChangeStatus = iota
	// Added signals that the file was modified
	Added
	// Deleted signals that the file was modified
	Deleted
)

const gitDiffStatusColumns = 2

// DiffNameStatus extracts the map of files and the action that was performed on them: added, modified or delete.
func DiffNameStatus(repoPath, previousCommit, currentCommit string) (map[string]FileChangeStatus, error) {
	m := make(map[string]FileChangeStatus)
	output, err := git(repoPath, "diff", "--no-renames", "--name-status", previousCommit, currentCommit)
	if err != nil {
		return nil, fmt.Errorf("error running diff: %s, %w", output, err)
	}
	// git diff --name-status returns plain text with 2 columns, the first is the
	// status and second file name, example:
	// M       tools/kaeter/README.md
	// A       tools/kaeter/git/diff.go
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		words := spacesRegex.Split(scanner.Text(), gitDiffStatusColumns)
		if len(words) != gitDiffStatusColumns {
			// Ignore lines with a different format
			continue
		}
		if mod, err := parseFileStatus(words[0]); err == nil {
			m[words[1]] = mod
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading the changed files: %w", err)
	}

	return m, nil
}

// parseFileStatus maps the git file filter statuses to internal FileChangeStatus.
// The statuses returned by git are documented on https://git-scm.com/docs/git-diff under --diff-filter.
func parseFileStatus(modifier string) (FileChangeStatus, error) {
	switch modifier {
	case "M":
		return Modified, nil
	case "A":
		return Added, nil
	case "D":
		return Deleted, nil
	default:
		return Modified, fmt.Errorf("could not parse git file change status %s", modifier)
	}
}

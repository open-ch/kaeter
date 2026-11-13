package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TreesExist reports, for each of paths, whether commit has a directory at that path. It runs a
// single git cat-file, so the cost doesn't grow with the number of paths.
//
//	git.TreesExist("path/to/repo", "HEAD", []string{"products", "products/foo"})
func TreesExist(repoPath, commit string, paths []string) (map[string]bool, error) {
	exists := make(map[string]bool, len(paths))
	if len(paths) == 0 {
		return exists, nil
	}
	var stdin bytes.Buffer
	for _, path := range paths {
		if strings.ContainsAny(path, "\n") {
			return nil, fmt.Errorf("path %q contains a newline", path)
		}
		fmt.Fprintf(&stdin, "%s:%s\n", commit, path)
	}
	// Every input line gets exactly one output line, in order: the object type, or
	// "<object> missing" when the path doesn't exist at commit.
	cmd := exec.CommandContext(context.Background(), "git", "cat-file", "--batch-check=%(objecttype)")
	cmd.Dir = repoPath
	cmd.Stdin = &stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running cat-file: %s, %w", stderr.String(), err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	i := 0
	for scanner.Scan() {
		if i >= len(paths) {
			return nil, fmt.Errorf("cat-file printed more lines than the %d paths asked for", len(paths))
		}
		exists[paths[i]] = scanner.Text() == "tree"
		i++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading cat-file output: %w", err)
	}
	if i != len(paths) {
		return nil, fmt.Errorf("cat-file printed %d lines for %d paths", i, len(paths))
	}
	return exists, nil
}

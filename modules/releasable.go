package modules

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
)

// ReleasableCommit is a commit that makes a module need a release.
type ReleasableCommit struct {
	SHA         string `json:"sha"`
	Subject     string `json:"subject"`
	AuthorEmail string `json:"authorEmail"`
}

// ReleasableCommitsQuery selects the commits GetReleasableCommits inspects.
type ReleasableCommitsQuery struct {
	// RepoRoot is the absolute path of the git repository.
	RepoRoot string
	// ModulePath is the module directory, relative to RepoRoot. Only commits touching it count,
	// kaeter dependencies are not considered.
	ModulePath string
	// FromRef is the commit of the latest release (exclusive). INIT yields no commits,
	// AUTORELEASE yields ErrAutoreleasePending, anything else must be a full commit hash.
	FromRef string
	// ToRef is the last commit to consider (inclusive), for example HEAD.
	ToRef string
	// IgnorePattern is a regexp matched against commit subjects, matching commits are skipped.
	// Usually the needsrelease.ignorepattern value of the repository's kaeter config, empty
	// disables it. Library callers must load it themselves; reading the config into the global
	// viper instance, as the kaeter CLI does, replaces the caller's own viper config.
	IgnorePattern string
}

// ErrAutoreleasePending is returned when the latest release of a module is still an AUTORELEASE.
var ErrAutoreleasePending = errors.New("autorelease pending")

// The git log format. Headers start with logHeaderMarker to tell them apart from file names,
// which can contain anything but NUL. The subject is the last field, so it may contain the
// separators.
const (
	logHeaderMarker   = "\x1e"
	logFieldSeparator = "\x1f"
	logFormat         = "%x1e%H%x1f%ae%x1f%s"
	logFieldCount     = 3
)

// GetReleasableCommits returns the non-merge commits in FromRef..ToRef that touch the module,
// oldest first. It skips commits whose subject matches IgnorePattern, and release-only commits:
// those whose changes under the module path are limited to versions.yaml (or versions.yml),
// CHANGELOG.md and files under kubernetes/. The latter keeps a release commit from making the module releasable
// again; any new file that a release writes inside the module must be added to that rule.
func GetReleasableCommits(query *ReleasableCommitsQuery) ([]ReleasableCommit, error) {
	if query.FromRef == InitRef {
		return nil, nil
	}
	if err := validateRefs(query.FromRef, query.ToRef); err != nil {
		return nil, err
	}
	ignoreRegex, err := regexp.Compile(query.IgnorePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid needsrelease ignore pattern %q: %w", query.IgnorePattern, err)
	}

	modulePath, err := normalizeModulePath(query.RepoRoot, query.ModulePath)
	if err != nil {
		return nil, err
	}

	revisionRange := fmt.Sprintf("%s..%s", query.FromRef, query.ToRef)
	gitlog, err := git.LogNameOnly(query.RepoRoot, revisionRange, logFormat, modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute git log on %s for %s: %w", modulePath, revisionRange, err)
	}

	commits := []ReleasableCommit{}
	err = parseLog(gitlog, func(commit ReleasableCommit, files []string) {
		switch {
		case query.IgnorePattern != "" && ignoreRegex.MatchString(commit.Subject):
			log.Debug("ignoring commit matching ignore pattern", "sha", commit.SHA, "subject", commit.Subject)
		case isReleaseOnlyChange(modulePath, files):
			log.Debug("ignoring release-only commit", "sha", commit.SHA, "subject", commit.Subject)
		default:
			commits = append(commits, commit)
		}
	})
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func validateRefs(fromRef, toRef string) error {
	switch {
	case fromRef == autoReleaseRef:
		return ErrAutoreleasePending
	case len(fromRef) != hashLength:
		return fmt.Errorf("invalid previous release ref %s", fromRef)
	case toRef == "":
		return errors.New("missing end ref")
	}
	return nil
}

func normalizeModulePath(repoRoot, modulePath string) (string, error) {
	if filepath.IsAbs(modulePath) {
		relPath, err := filepath.Rel(repoRoot, modulePath)
		if err != nil {
			return "", fmt.Errorf("module path %s is not in repository %s: %w", modulePath, repoRoot, err)
		}
		modulePath = relPath
	}
	modulePath = filepath.ToSlash(filepath.Clean(modulePath))
	if modulePath == ".." || strings.HasPrefix(modulePath, "../") {
		return "", fmt.Errorf("module path %s is outside of repository %s", modulePath, repoRoot)
	}
	return modulePath, nil
}

// parseLog calls handle for each commit of git.LogNameOnly output written with logFormat.
// Output before the first header, such as git warnings, is skipped.
func parseLog(gitlog string, handle func(commit ReleasableCommit, files []string)) error {
	var commit *ReleasableCommit
	var files []string
	afterHeader := false
	for _, token := range strings.Split(gitlog, "\x00") {
		if afterHeader {
			// git separates the header from the file list with a newline.
			token = strings.TrimPrefix(token, "\n")
		}
		afterHeader = false
		if header, ok := strings.CutPrefix(token, logHeaderMarker); ok {
			if commit != nil {
				handle(*commit, files)
			}
			fields := strings.SplitN(header, logFieldSeparator, logFieldCount)
			if len(fields) != logFieldCount {
				return fmt.Errorf("unexpected git log output: %q", token)
			}
			commit = &ReleasableCommit{SHA: fields[0], AuthorEmail: fields[1], Subject: fields[2]}
			files = nil
			afterHeader = true
			continue
		}
		if commit != nil && token != "" {
			files = append(files, token)
		}
	}
	if commit != nil {
		handle(*commit, files)
	}
	return nil
}

// isReleaseOnlyChange reports whether files, paths relative to the repository root, only
// contain release metadata of the module at modulePath. An empty list is not release-only.
func isReleaseOnlyChange(modulePath string, files []string) bool {
	if len(files) == 0 {
		return false
	}
	versionsFiles := []string{path.Join(modulePath, "versions.yaml"), path.Join(modulePath, "versions.yml")}
	changelogFile := path.Join(modulePath, "CHANGELOG.md")
	kubernetesDir := path.Join(modulePath, "kubernetes") + "/"
	for _, file := range files {
		if !slices.Contains(versionsFiles, file) && file != changelogFile && !strings.HasPrefix(file, kubernetesDir) {
			return false
		}
	}
	return true
}

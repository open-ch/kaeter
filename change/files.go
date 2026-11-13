package change

import (
	"fmt"
	"path"
	"slices"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
)

// Files the list of add, modified and deleted files
type Files struct {
	Added       []string
	Removed     []string
	Modified    []string
	Directories Directories
}

// Directories lists the directories the change creates or deletes, at every depth: adding
// a/b/file where a/ didn't exist lists both a and a/b. Git tracks files only, so a directory
// is created by its first file and deleted with its last one.
type Directories struct {
	Added   []string
	Removed []string
}

// FileCheck reads the git changes between two commit and compiles a list of added, changed and deleted files
func (d *Detector) FileCheck(_ *Information) (files Files, err error) {
	fileChanges, err := git.DiffNameStatus(d.RootPath, d.PreviousCommit, d.CurrentCommit)
	if err != nil {
		return files, fmt.Errorf("files detector unable to perform git diff: %w", err)
	}
	files.Added = make([]string, 0)
	files.Modified = make([]string, 0)
	files.Removed = make([]string, 0)
	for file, modifier := range fileChanges {
		switch modifier {
		case git.Modified:
			files.Modified = append(files.Modified, file)
		case git.Added:
			files.Added = append(files.Added, file)
		case git.Deleted:
			files.Removed = append(files.Removed, file)
			// Note: we considered adding a case to cover renamed, traditionally however a rename consisits of added and deleted.
			//       Further more, when a file is renamed but include a lot of changes it will often show as added/deleted,
			//       so it's not clear how we can add it in a backwards compatible way and maintain consistent results.
		default:
			// DiffNameStatus reports only added, modified and deleted files.
		}
	}
	slices.Sort(files.Added)
	slices.Sort(files.Modified)
	slices.Sort(files.Removed)

	// A new directory exists at the current commit only, and holds an added file. A deleted one
	// exists at the previous commit only, and held a removed file.
	files.Directories.Added, err = d.directoriesMissingAt(d.PreviousCommit, files.Added)
	if err != nil {
		return files, err
	}
	files.Directories.Removed, err = d.directoriesMissingAt(d.CurrentCommit, files.Removed)
	if err != nil {
		return files, err
	}

	log.Debug("DetectorFiles",
		"Modified", files.Modified,
		"Added", files.Added,
		"Deleted", files.Removed,
		"AddedDirectories", files.Directories.Added,
		"RemovedDirectories", files.Directories.Removed)

	return files, nil
}

// AllFiles returns a slice all files that were added, modified or removed
func (f *Files) AllFiles() []string {
	allFiles := make([]string, 0, len(f.Added)+len(f.Modified)+len(f.Removed))
	allFiles = append(allFiles, f.Added...)
	allFiles = append(allFiles, f.Modified...)
	allFiles = append(allFiles, f.Removed...)
	return allFiles
}

// directoriesMissingAt returns, sorted, the parent directories of filePaths, at every depth, that
// don't exist at commit.
func (d *Detector) directoriesMissingAt(commit string, filePaths []string) ([]string, error) {
	candidates := parentDirectories(filePaths)
	exists, err := git.TreesExist(d.RootPath, commit, candidates)
	if err != nil {
		return nil, fmt.Errorf("files detector unable to list the directories at %s: %w", commit, err)
	}
	missing := make([]string, 0)
	for _, dir := range candidates {
		if !exists[dir] {
			missing = append(missing, dir)
		}
	}
	return missing, nil
}

// parentDirectories returns, sorted and deduplicated, every parent directory of filePaths.
func parentDirectories(filePaths []string) []string {
	seen := make(map[string]bool)
	dirs := make([]string, 0)
	for _, filePath := range filePaths {
		for dir := path.Dir(filePath); dir != "." && dir != "/" && !seen[dir]; dir = path.Dir(dir) {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	slices.Sort(dirs)
	return dirs
}

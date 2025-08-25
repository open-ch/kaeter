package change

import (
	"fmt"
	"sort"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
)

// Files the list of add, modified and deleted files
type Files struct {
	Added    []string
	Removed  []string
	Modified []string
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
		}
	}
	sort.Strings(files.Added)
	sort.Strings(files.Modified)
	sort.Strings(files.Removed)

	log.Debug("DetectorFiles",
		"Modified", files.Modified,
		"Added", files.Added,
		"Deleted", files.Removed)

	return files, nil
}

// AllFiles returns a slice all files that were added, modified or removed
func (f Files) AllFiles() []string {
	allFiles := make([]string, 0, len(f.Added)+len(f.Modified)+len(f.Removed))
	allFiles = append(allFiles, f.Added...)
	allFiles = append(allFiles, f.Modified...)
	allFiles = append(allFiles, f.Removed...)
	return allFiles
}

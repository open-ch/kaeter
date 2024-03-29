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
			// TODO consider adding a case to cover renamed (different from modified but would need to be somehow backwards compatible...)
		}
	}
	sort.Strings(files.Added)
	sort.Strings(files.Modified)
	sort.Strings(files.Removed)

	log.Debugf("DetectorFiles: Modified: %v", files.Modified)
	log.Debugf("DetectorFiles: Added: %v", files.Added)
	log.Debugf("DetectorFiles: Deleted: %v", files.Removed)

	return files, nil
}

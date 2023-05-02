package change

import (
	"os"
	"sort"

	"github.com/open-ch/go-libs/gitshell"

	"github.com/open-ch/kaeter/kaeter/log"
)

// Files the list of add, modified and deleted files
type Files struct {
	Added    []string
	Removed  []string
	Modified []string
}

// FileCheck reads the git changes between two commit and compiles a list of added, changed and deleted files
func (d *Detector) FileCheck(_ *Information) (files Files) {
	fileChanges, err := gitshell.GitFileDiff(d.RootPath, d.PreviousCommit, d.CurrentCommit)
	if err != nil {
		// TODO return error rather than exit
		log.Errorf("DetectorFiles: Unable to perform git diff: %s", err)
		os.Exit(1)
	}
	files.Added = make([]string, 0)
	files.Modified = make([]string, 0)
	files.Removed = make([]string, 0)
	for file, modifier := range fileChanges {
		switch modifier {
		case gitshell.Modified:
			files.Modified = append(files.Modified, file)
		case gitshell.Added:
			files.Added = append(files.Added, file)
		case gitshell.Deleted:
			files.Removed = append(files.Removed, file)
		}
	}
	sort.Strings(files.Added)
	sort.Strings(files.Modified)
	sort.Strings(files.Removed)

	log.Debugf("DetectorFiles: Modified: %v", files.Modified)
	log.Debugf("DetectorFiles: Added: %v", files.Added)
	log.Debugf("DetectorFiles: Deleted: %v", files.Removed)

	return files
}

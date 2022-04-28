package change

import (
	"github.com/sirupsen/logrus"
)

// Detector contains the configuration of the change detector
type Detector struct {
	Logger         *logrus.Logger
	RootPath       string
	PreviousCommit string
	CurrentCommit  string
}

// Information contains the summary of all changes
type Information struct {
	Files  Files
	Commit CommitMsg
	Bazel  BazelChange
	Kaeter KaeterChange
	Helm   HelmChange
}

// Check performs the change detection over all modules
func (d *Detector) Check(skipBazel bool) (info *Information) {
	info = new(Information)

	// Note that order matters here as some checkers use results of the previous:
	info.Commit = d.CommitCheck(info)
	info.Files = d.FileCheck(info)
	if skipBazel {
		d.Logger.Info("Skipping bazel check")
	} else {
		info.Bazel = d.BazelCheck(info)
	}
	info.Kaeter = d.KaeterCheck(info)
	info.Helm = d.HelmCheck(info)

	return info
}

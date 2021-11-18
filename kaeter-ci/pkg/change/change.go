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

// New initialises a new Detector
func New(level logrus.Level, rootPath, previousCommit, currentCommit string) *Detector {
	d := Detector{
		Logger:         logrus.New(),
		RootPath:       rootPath,
		PreviousCommit: previousCommit,
		CurrentCommit:  currentCommit,
	}

	d.Logger.Level = level

	return &d
}

// Information contains the summary of all changes
type Information struct {
	Files  Files
	Bazel  BazelChange
	Kaeter KaeterChange
	Helm   HelmChange
}

// Check performs the change detection over all modules
func (d *Detector) Check() (info *Information) {
	info = new(Information)

	// Note that order matters here as some checkers use results of the previous:
	info.Files = d.FileCheck(info)
	info.Bazel = d.BazelCheck(info)
	info.Kaeter = d.KaeterCheck(info)
	info.Helm = d.HelmCheck(info)

	return info
}

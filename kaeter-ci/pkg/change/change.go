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
	PullRequest    *PullRequest
}

// Information contains the summary of all changes
type Information struct {
	Files       Files
	Commit      CommitMsg
	Kaeter      KaeterChange
	Helm        HelmChange
	PullRequest *PullRequest
	// ref: https://pkg.go.dev/encoding/json#Marshal
}

// PullRequest can hold optional informations if a pull request
// is open on the vcs hosting platform
type PullRequest struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// Check performs the change detection over all modules
func (d *Detector) Check() (info *Information) {
	info = new(Information)

	// Note that order matters here as some checkers use results of the previous:
	info.Commit = d.CommitCheck(info)
	info.Files = d.FileCheck(info)
	info.Kaeter = d.KaeterCheck(info)
	info.Helm = d.HelmCheck(info)
	info.PullRequest = d.PullRequest

	return info
}

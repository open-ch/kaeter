package change

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/modules"
)

// Detector contains the configuration of the change detector
type Detector struct {
	RootPath       string
	PreviousCommit string
	CurrentCommit  string
	KaeterModules  []modules.KaeterModule
	PullRequest    *PullRequest
}

// Information contains the summary of all changes
type Information struct {
	Files       Files
	Commit      CommitMsg
	Kaeter      KaeterChange
	Helm        HelmChange
	PullRequest *PullRequest `json:",omitempty"`
	// ref: https://pkg.go.dev/encoding/json#Marshal
}

// PullRequest can hold optional informations if a pull request
// is open on the vcs hosting platform
type PullRequest struct {
	Title       string               `json:"title,omitempty"`
	Body        string               `json:"body,omitempty"`
	ReleasePlan *actions.ReleasePlan `json:",omitempty"`
}

// Check performs the change detection over all modules
func (d *Detector) Check() (*Information, error) {
	info := &Information{}

	// Note that order matters here as some checkers use results of the previous:
	info.PullRequest = d.PullRequestCommitCheck(info)
	commitInfo, err := d.CommitCheck(info)
	if err != nil {
		return info, err
	}
	info.Commit = commitInfo

	// TODO run the next steps in parallel if they do not
	// need each other for better performance.
	fileChanges, err := d.FileCheck(info)
	if err != nil {
		return info, err
	}
	info.Files = fileChanges

	katerChange, err := d.KaeterCheck(info)
	if err != nil {
		return info, err
	}
	info.Kaeter = katerChange

	info.Helm = d.HelmCheck(info)

	return info, nil
}

// LoadChangeset reads changeset.json back into Information.
func LoadChangeset(changesetPath string) (*Information, error) {
	bytes, err := os.ReadFile(changesetPath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", changesetPath, err)
	}

	info := &Information{}
	//nolint:musttag
	if err := json.Unmarshal(bytes, &info); err != nil {
		return nil, fmt.Errorf("could not parse %s: %w", changesetPath, err)
	}

	return info, nil
}

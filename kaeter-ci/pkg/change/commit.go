package change

import (
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"regexp"
	"strings"

	"github.com/open-ch/go-libs/gitshell"
)

// CommitMsg contains the list of commit message tags
type CommitMsg struct {
	Tags        []string
	ReleasePlan *kaeter.ReleasePlan `json:",omitempty"`
}

// In order to match every tag separately in a concise expression,
// we would need to use backreferences which are not supported by regexp
// for reasons of efficiency; we repeat the expression three times instead.
// It's not the prettiest thing but it's simple enough and it works in order
// to avoid introducing a new 3rd party library that would support our usecase.
// The main expression is \[([a-z0-9]{1,24})\] and (?:) is used so that the
// additional tags are not counted as separate submatches.
var tagRegex = regexp.MustCompile(
	`(?:\[([a-z0-9]{1,24})\])(?:\[([a-z0-9]{1,24})\])?(?:\[([a-z0-9]{1,24})\])?`)

// CommitCheck returns information about the current commit message
// (current as defined by the parameters, not necessarily HEAD).
// This includes the first 3 [tags] in the subject line,
// the release plan if one is available.
func (d *Detector) CommitCheck(changes *Information) (c CommitMsg) {
	d.Logger.Infof("Root path: %s", d.RootPath)
	currentCommitMsg, err := gitshell.GitCommitMessageFromHash(d.RootPath, d.CurrentCommit)
	if err != nil {
		d.Logger.Fatalf("Failed to get commit message for %s (%s): %s", d.CurrentCommit, err, currentCommitMsg)
	}

	capturedTags := extractTags(currentCommitMsg)
	if len(capturedTags) > 0 {
		c.Tags = capturedTags
		d.Logger.Debugf("Captured tags are: " + strings.Join(c.Tags, ","))
	} else {
		d.Logger.Debugf("No tags specified in current commit message:\n%s", currentCommitMsg)
	}

	releasePlan, err := kaeter.ReleasePlanFromCommitMessage(currentCommitMsg)
	if err != nil {
		d.Logger.Debugf("No release plan: %s", err)
		c.ReleasePlan = &kaeter.ReleasePlan{[]kaeter.ReleaseTarget{}}
	} else {
		c.ReleasePlan = releasePlan
	}

	return
}

func extractTags(commitMessage string) []string {
	capturedTags := tagRegex.FindStringSubmatch(commitMessage)

	if len(capturedTags) > 0 {
		// Regex explicitly defines at least one submatch,
		// so if the array exists it is > 1
		return removeTrailingEmptyStrings(capturedTags[1:])
	}

	return []string{}
}

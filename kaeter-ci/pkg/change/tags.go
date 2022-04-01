package change

import (
	"regexp"
	"strings"

	"github.com/open-ch/go-libs/gitshell"
)

// CommitMsg contains the list of commit message tags
type CommitMsg struct {
	Tags []string
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

// TagCheck returns the different flags present in the commit message in Tags
func (d *Detector) TagCheck(changes *Information) (c CommitMsg) {
	currentCommitMsg := gitshell.GitCommitMessageFromHash(d.RootPath, d.CurrentCommit)

	capturedTags := tagRegex.FindStringSubmatch(currentCommitMsg)

	if len(capturedTags) > 0 {
		// Regex explicitly defines at least one submatch,
		// so if the array exists it is > 1
		c.Tags = removeTrailingEmptyStrings(capturedTags[1:])
		d.Logger.Debugf("Captured tags are: " + strings.Join(c.Tags, ","))
	} else {
		d.Logger.Debugf("No tags specified in current commit message:\n%s", currentCommitMsg)
	}

	return
}

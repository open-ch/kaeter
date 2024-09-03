package change

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
)

// CommitMsg contains the list of commit message tags
type CommitMsg struct {
	Tags        []string
	ReleasePlan *actions.ReleasePlan `json:",omitempty"`
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
func (d *Detector) CommitCheck(_ *Information) (c CommitMsg, e error) {
	currentCommitMsg, err := git.GetCommitMessageFromRef(d.RootPath, d.CurrentCommit)
	if err != nil {
		return c, fmt.Errorf("failed to get commit message for %s: %s\n%w", d.CurrentCommit, currentCommitMsg, err)
	}

	capturedTags := extractTags(currentCommitMsg)
	if len(capturedTags) > 0 {
		c.Tags = capturedTags
		log.Debug("Tags extracted from commit message", "tags", strings.Join(c.Tags, ","))
	} else {
		log.Debug("No tags specified in current commit message", "commitMessage", currentCommitMsg)
	}

	releasePlan, err := actions.ReleasePlanFromCommitMessage(currentCommitMsg)
	if err != nil {
		log.Debug("No release plan", "error", err)
		c.ReleasePlan = &actions.ReleasePlan{Releases: []actions.ReleaseTarget{}}
	} else {
		c.ReleasePlan = releasePlan
	}

	return c, nil
}

// PullRequestCommitCheck allows checking for a release to be from a pull
// request assuming:
// - The PR has a title and body
// - The title and body combined will beocome the merged commit message
// This in turns allows loading kaeter release plans on what is to be released.
func (d *Detector) PullRequestCommitCheck(_ *Information) (pr *PullRequest) {
	pr = &PullRequest{
		Title: d.PullRequest.Title,
		Body:  d.PullRequest.Body,
	}
	assumedCommitMessage := fmt.Sprintf("%s\n%s", pr.Title, pr.Body)
	log.Debug("Extracting release plan from PR data", "commitMessage", assumedCommitMessage)

	releasePlan, err := actions.ReleasePlanFromCommitMessage(assumedCommitMessage)
	if err != nil {
		log.Debug("No release plan found in PR", "error", err)
		pr.ReleasePlan = &actions.ReleasePlan{Releases: []actions.ReleaseTarget{}}
	} else {
		pr.ReleasePlan = releasePlan
	}

	return pr
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

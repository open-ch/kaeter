package ci

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/change"
)

// AutoReleaseConfig allows customizing how the kaeter release
// will handle the process
type AutoReleaseConfig struct {
	ChangesetPath       string
	PullRequestBodyPath string
	Logger              *logrus.Logger
}

const (
	// AutoReleasePlanPrefix marks the start of the auto release
	// block in PRs. It is added and used to search for and remove previous
	// autorelease blocks.
	AutoReleasePlanPrefix = "#### Autorelease Plan:"
	// AutoReleasePlanSuffix marks the end of the auto release
	// block in PRs. It is added and used to search for and remove previous
	// autorelease blocks.
	AutoReleasePlanSuffix    = "(release plan auto generated by CI)"
	regexFindAllNoCountLimit = -1
)

// Matching an existing release plan:
// - (?s) flags for the regex
//   - s = . matches \n
//
// - Line that starts with prefix
// - Anything in between (greedy)
// - Line that ends with suffix
// - One+ newline(s) after the suffix to avoid leading gap
// - global to avoid recompmiling regex each time function is called
//
// Notes:
// The JSON encoded PRs from github use CRLFs as line breaks so we can't use
// the multiline (m) flag on go regex but use (\r?\n) instead.
// If the gready match becomes a performance issue we could instead match
// start index, end index and then strip from start to end using string
// operations. However given the succintness of some pull request
// bodies this optimization might not be justified.
//
// See also https://golang.org/s/re2syntax for details on go regex syntax.
var prPlanRegexp = regexp.MustCompile(fmt.Sprintf(
	"(?s)(\r?\n)*%s(\r?\n).+(\r?\n)%s(\r?\n)*",
	regexp.QuoteMeta(AutoReleasePlanPrefix),
	regexp.QuoteMeta(AutoReleasePlanSuffix),
))

// GetUpdatedPRBody reads the changeset and generates a new PR body
// which includes the autorelease plan based on the changes at the top.
func (arc *AutoReleaseConfig) GetUpdatedPRBody() error {
	changeset, err := change.LoadChangeset(arc.ChangesetPath)
	if err != nil {
		return fmt.Errorf("could not load changeset: %w", err)
	}
	arc.Logger.Debugf("kaeter ci: changeset... %v", changeset)

	if len(changeset.Commit.ReleasePlan.Releases) > 0 {
		return errors.New("prepare release detected: incompatible release(s)")
	}

	arc.Logger.Infof("kaeter ci: Pull request body (before):\n%s", changeset.PullRequest.Body)

	autoreleaseplan, err := getAutoReleasePlan(changeset)
	if err != nil {
		return fmt.Errorf("could not generate release plan: %w", err)
	}
	arc.Logger.Infof("kaeter ci: Autorelease plan (current):\n%s", autoreleaseplan)

	cleanPRBody := stripAutoReleasePlan(changeset.PullRequest.Body)
	newPRBody := insertPlan(cleanPRBody, autoreleaseplan)
	arc.Logger.Infof("kaeter ci: Pull request body (updated):\n%s", newPRBody)

	err = os.WriteFile(arc.PullRequestBodyPath, []byte(newPRBody), 0644)
	if err != nil {
		return fmt.Errorf("could not write pull request body to file %s: %w", arc.PullRequestBodyPath, err)
	}
	arc.Logger.Infof("kaeter ci: saved pr body to %s", arc.PullRequestBodyPath)

	return nil
}

func getAutoReleasePlan(changeset *change.Information) (string, error) {
	// To avoid inconsistant/unstable output plan we sort by
	// keys and filter out non autoreleases then iterate over modules.
	var idsOfModulesWithAutorelease []string
	for moduleID := range changeset.Kaeter.Modules {
		if changeset.Kaeter.Modules[moduleID].AutoRelease != "" {
			idsOfModulesWithAutorelease = append(idsOfModulesWithAutorelease, moduleID)
		}
	}

	if len(idsOfModulesWithAutorelease) < 1 {
		return "", nil
	}

	sort.Strings(idsOfModulesWithAutorelease)

	var planBuilder strings.Builder
	_, err := planBuilder.WriteString(AutoReleasePlanPrefix + "\n")
	if err != nil {
		return "", err
	}

	for _, moduleID := range idsOfModulesWithAutorelease {
		_, err = planBuilder.WriteString(fmt.Sprintf("- %s:%s\n", moduleID, changeset.Kaeter.Modules[moduleID].AutoRelease))
		if err != nil {
			return "", err
		}
	}

	_, err = planBuilder.WriteString("\n" + AutoReleasePlanSuffix)
	if err != nil {
		return "", err
	}

	return planBuilder.String(), nil
}

func stripAutoReleasePlan(body string) string {
	matches := prPlanRegexp.FindAllString(body, regexFindAllNoCountLimit)

	if matches == nil || len(matches) < 1 {
		return body
	}

	// We use TrimSpace to make sure we don't add leading or trailing new lines.
	return strings.TrimSpace(prPlanRegexp.ReplaceAllString(body, "\n"))
	// Note: we could add additional sanity checks after stripping:
	//       - Check if any AutoReleasePlanPrefix still found
	//       - Check if any AutoReleasePlanSuffix still found
	//       and raise errors that PR body needs manual cleanup.
}

func insertPlan(body, plan string) string {
	if plan == "" {
		return body
	}

	return fmt.Sprintf("%s\n%s", body, plan)
}

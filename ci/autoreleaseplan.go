package ci

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/open-ch/kaeter/change"
	"github.com/open-ch/kaeter/log"
)

// AutoReleaseConfig allows customizing how the kaeter release
// will handle the process
type AutoReleaseConfig struct {
	ChangesetPath       string
	PullRequestBodyPath string
}

const (
	// AutoReleasePlanPrefix marks the start of the auto release
	// block in PRs. It is added and used to search for and remove previous
	// autorelease blocks.
	AutoReleasePlanPrefix    = "Autorelease-Plan"
	regexFindAllNoCountLimit = -1
)

// Matching an existing autorelease plan:
// - Line that starts with prefix
// - Anything in between
// - One newline
// See also https://golang.org/s/re2syntax for details on go regex syntax.
var prPlanRegexp = regexp.MustCompile(fmt.Sprintf(
	"(?m)^%s: .+$\r?\n?",
	regexp.QuoteMeta(AutoReleasePlanPrefix),
))

// GetUpdatedPRBody reads the changeset and generates a new PR body
// which includes the autorelease plan based on the changes at the top.
func (arc *AutoReleaseConfig) GetUpdatedPRBody() error {
	changeset, err := change.LoadChangeset(arc.ChangesetPath)
	if err != nil {
		return fmt.Errorf("could not load changeset: %w", err)
	}
	log.Debug("kaeter ci: changeset loaded", "changelset", changeset)

	if len(changeset.Commit.ReleasePlan.Releases) > 0 {
		return errors.New("prepare release detected: incompatible release(s)")
	}

	log.Info("kaeter ci: Original pull request body loaded", "body", changeset.PullRequest.Body)

	autoreleaseplan, err := getAutoReleasePlan(changeset)
	if err != nil {
		return fmt.Errorf("could not generate release plan: %w", err)
	}
	log.Info("kaeter ci: new plan generated", "autoreleaseplan", autoreleaseplan)

	cleanPRBody := stripAutoReleasePlan(changeset.PullRequest.Body)
	newPRBody := insertPlan(cleanPRBody, autoreleaseplan)
	log.Info("kaeter ci: Updated pull request body generated", "body", newPRBody)

	err = os.WriteFile(arc.PullRequestBodyPath, []byte(newPRBody), 0600)
	if err != nil {
		return fmt.Errorf("could not write pull request body to file %s: %w", arc.PullRequestBodyPath, err)
	}
	log.Info("kaeter ci: saved pr body", "path", arc.PullRequestBodyPath)

	return nil
}

func getAutoReleasePlan(changeset *change.Information) (string, error) {
	// To avoid inconsistent/unstable output plan we sort by
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
	_, err := planBuilder.WriteString("\n")
	if err != nil {
		return "", err
	}
	for _, moduleID := range idsOfModulesWithAutorelease {
		_, err := planBuilder.WriteString(fmt.Sprintf("%s: %s:%s\n", AutoReleasePlanPrefix, moduleID, changeset.Kaeter.Modules[moduleID].AutoRelease))
		if err != nil {
			return "", err
		}
	}

	return planBuilder.String(), nil
}

func stripAutoReleasePlan(body string) string {
	matches := prPlanRegexp.FindAllString(body, regexFindAllNoCountLimit)

	if len(matches) < 1 {
		return body
	}

	// We use TrimSpace to make sure we don't add leading or trailing new lines.
	return strings.TrimSpace(prPlanRegexp.ReplaceAllString(body, ""))
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

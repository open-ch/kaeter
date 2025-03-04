package modules

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
)

const hashLength = 40
const autoReleaseRef = "AUTORELEASE"

// InitRef is the commit ref value used for initialize entries in kaeter modules.
const InitRef = "INIT"

// NeedsReleaseInfo info about a modules latest release and unreleased changes
type NeedsReleaseInfo struct {
	ModuleID   string `json:"moduleId"`
	ModulePath string `json:"modulePath"`
	// LatestReleaseTimestamp is based on the contents of the versions.yaml,
	// and it will be nil if the module did not have a release.
	LatestReleaseTimestamp *time.Time `json:"latestReleaseTimestamp"` // note: nill if no releases
	// Number of unreleased commits in the module itself since the latest release
	// if there is a pending AUTORELEASE this number is based on the release
	// before that up to HEAD.
	UnreleasedCommitCount           int  `json:"unreleasedCommitCount"`
	UnreleasedDependencyCommitCount int  `json:"unreleasedDependencyCommitCount"`
	AutoreleasePending              bool `json:"autoreleasePending"`
	// Errors contains details about any failures to load detailed release
	// info for this module. If error is set the fields above might or might not
	// be set depending on when the errors happened.
	Error error `json:"-"`
	// ErrorStr is the string value of the error which we map to error when encoding
	// to json to avoid serialization issues.
	ErrorStr string `json:"error,omitempty"`
}

// Style definitions.
//
//nolint:gochecknoglobals,mnd // Can't declare these objects as const
var (
	highlight   = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special     = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	errorOrange = lipgloss.Color("#f96616")
	errorRed    = lipgloss.Color("#f91638")

	errorBox = lipgloss.NewStyle().
			Bold(true).
			Foreground(errorOrange).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(errorRed).
			PaddingLeft(4).
			Width(80)
	moduleBox = lipgloss.NewStyle().
			Foreground(special).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			PaddingLeft(4).
			Width(80)

	moduleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)
)

var (
	errAutoreleasePending = errors.New("autorelease pending")
)

// PrintModuleInfo outputs info about a module at the given relative path
// in pretty format, if said module exists.
func PrintModuleInfo(path string) {
	versions, err := loadModule(path)
	if err != nil {
		_, _ = fmt.Println(errorBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			"Unable to load module at", path,
			"because", err.Error(),
		)))
		return
	}

	latestRelease := getLatestRelease(versions.ReleasedVersions)
	releaseDate := "never"
	estReleaseAge := "∞"
	if latestRelease.CommitID != InitRef {
		releaseDate = latestRelease.Timestamp.Format(time.DateTime)
		estReleaseAge = fmt.Sprintf("%.f", time.Since(latestRelease.Timestamp).Hours()/24) //nolint:mnd
	}
	unreleasedChanges := getUnreleasedChangesSummary(latestRelease.CommitID, path)

	_, _ = fmt.Println(moduleBox.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			moduleHeader.Render("Module ID: ", versions.ID),
			fmt.Sprintf("Path: %s", path),
			fmt.Sprintf("Releases: %d", len(versions.ReleasedVersions)-1), // Ignore the 0.0.0 INIT version
			fmt.Sprintf("Current release: %s", latestRelease.Number),
			fmt.Sprintf("Released: %s ~%s days ago", releaseDate, estReleaseAge),
			fmt.Sprintf("Unreleased changes:\n%s", unreleasedChanges),
		),
	))
}

// GetNeedsReleaseInfoIn will first detect all module paths, if this fails
// an error will be returned, then it will emit the processing results of each
// modules (including errors) on the channel
func GetNeedsReleaseInfoIn(path string) (chan NeedsReleaseInfo, error) {
	modulesChan := make(chan NeedsReleaseInfo)
	absPath, err := filepath.Abs(path)
	if err != nil {
		close(modulesChan)
		return modulesChan, fmt.Errorf("unable to convert %s to absolute path, cannot detect modules: %w", path, err)
	}

	go func() {
		defer close(modulesChan)

		resultsChan := streamFoundIn(absPath)

		for result := range resultsChan {
			if err != nil {
				modulesChan <- NeedsReleaseInfo{
					Error:    err,
					ErrorStr: err.Error(),
				}
				continue
			}
			modulesChan <- getModuleNeedsReleaseInfo(result.module)
		}
	}()
	return modulesChan, nil
}

func getLatestRelease(releasedVersions []*VersionMetadata) *VersionMetadata {
	lastEntry := releasedVersions[len(releasedVersions)-1]
	if lastEntry.CommitID == autoReleaseRef && len(releasedVersions) > 1 {
		// Return the one before last if the last is a pending autorelease
		return releasedVersions[len(releasedVersions)-2]
	}
	return lastEntry
}

func hasPendingAutorelease(releasedVersions []*VersionMetadata) bool {
	lastReleaseEntry := releasedVersions[len(releasedVersions)-1]
	return lastReleaseEntry.CommitID == autoReleaseRef
}

func getUnreleasedChangesSummary(previousReleaseRef, path string) string {
	switch {
	case previousReleaseRef == autoReleaseRef:
		return "yes, AUTORELEASE pending."
	case previousReleaseRef == InitRef:
		return "Module never had a release. Everything is a change!"
	}

	gitlog, err := getUnreleasedCommitsLog(previousReleaseRef, path)
	if err != nil {
		log.Error("Error running git log", "error", err)
		return fmt.Sprintf("error: Unable to fetch changes since last release (%s)", err)
	}
	return gitlog
}

func getUnreleasedCommitsLog(previousReleaseRef string, paths ...string) (string, error) {
	switch {
	case previousReleaseRef == autoReleaseRef:
		return "", errAutoreleasePending
	case previousReleaseRef == InitRef:
		return "", nil
	case len(previousReleaseRef) != hashLength:
		return "", fmt.Errorf("invalid previous release ref %s", previousReleaseRef)
	}

	repoRoot := viper.GetString("repoRoot")
	revisionRange := fmt.Sprintf("%s..HEAD", previousReleaseRef)
	// TODO can we move this to a struct/interface? that would make mocking possible and testing easier
	// However likely still need a mock kaeter module for other parts of this so it's only relatively easier...
	gitlog, err := git.LogOneLine(repoRoot, revisionRange, paths...)
	if err != nil {
		return "", fmt.Errorf("failed to compute git log on %s since %s: %w", paths, revisionRange, err)
	}

	return gitlog, nil
}

func loadModule(path string) (*Versions, error) {
	absVersionsPath, err := GetVersionsFilePath(path)
	if err != nil {
		return nil, err
	}

	versions, err := ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func getModuleNeedsReleaseInfo(moduleInfo *KaeterModule) NeedsReleaseInfo {
	latestRelease := getLatestRelease(moduleInfo.versions.ReleasedVersions)
	latestReleaseTimestamp := &latestRelease.Timestamp
	if latestRelease.CommitID == InitRef {
		latestReleaseTimestamp = nil
	}

	var infoErrs error

	commitLog, err := getUnreleasedCommitsLog(latestRelease.CommitID, moduleInfo.ModulePath)
	commitCount := countUnreleasedCommits(commitLog)
	if err != nil {
		infoErrs = errors.Join(infoErrs, fmt.Errorf("unable to generate unreleased commit count for module %w", err))
		commitCount = -1
	}
	dependenciesCommitCount := 0
	if len(moduleInfo.versions.Dependencies) > 0 {
		depsCommitLog, err := getUnreleasedCommitsLog(latestRelease.CommitID, moduleInfo.versions.Dependencies...)
		dependenciesCommitCount = countUnreleasedCommits(depsCommitLog)
		if err != nil {
			infoErrs = errors.Join(infoErrs, fmt.Errorf("unable to generate commit count for dependencies %w", err))
			dependenciesCommitCount = -1
		}
	}

	errorStr := ""
	if infoErrs != nil {
		errorStr = infoErrs.Error()
	}
	return NeedsReleaseInfo{
		ModuleID:                        moduleInfo.versions.ID,
		ModulePath:                      moduleInfo.ModulePath,
		LatestReleaseTimestamp:          latestReleaseTimestamp,
		UnreleasedCommitCount:           commitCount,
		UnreleasedDependencyCommitCount: dependenciesCommitCount,
		AutoreleasePending:              hasPendingAutorelease(moduleInfo.versions.ReleasedVersions),
		Error:                           infoErrs,
		ErrorStr:                        errorStr,
	}
}

func countUnreleasedCommits(commitLog string) int {
	ignorePattern := viper.GetString("needsrelease.ignorepattern")
	commitCount := 0

	// Rather than ignoring/skipping empty lines later trim the output before starting
	cleanLog := strings.Trim(commitLog, "\n\t ")
	if cleanLog == "" {
		return 0
	}
	lines := strings.Split(cleanLog, "\n")
	for _, line := range lines {
		if ignorePattern != "" && strings.Contains(line, ignorePattern) {
			log.Debug("ignoring commit", "commitlogline", line)
			continue
		}
		commitCount++
	}
	return commitCount
}

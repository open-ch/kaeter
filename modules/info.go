package modules

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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

// ModuleNeedsReleaseInfo info about a modules latest release and unreleased changes
type ModuleNeedsReleaseInfo struct {
	ModuleID               string     `json:"moduleId"`
	ModulePath             string     `json:"modulePath"`
	LatestReleaseTimestamp *time.Time `json:"latestReleaseTimestamp"` // note: nill if no releases
	UnreleasedCommitsCount int        `json:"unreleasedCommitsCount"`
	AutoreleasePending     bool       `json:"autoreleasePending"`
	Error                  error      `json:"moduleParsingErrors,omitempty"`
}

type moduleInfo struct {
	moduleAbsolutePath       string
	moduleRelativePath       string
	versions                 *Versions
	versionsYamlAbsolutePath string
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
	unreleasedChanges := getUnreleasedChangesSummary(path, latestRelease.CommitID)

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
func GetNeedsReleaseInfoIn(path string) (chan ModuleNeedsReleaseInfo, error) {
	modulesChan := make(chan ModuleNeedsReleaseInfo)

	// TODO standardize module loading or reuse module inventory here
	versionsFiles, err := getSortedModulesFoundInPath(path)
	if err != nil {
		close(modulesChan)
		return modulesChan, err
	}

	go func() {
		defer close(modulesChan)

		for _, versionsYamlPath := range versionsFiles {
			moduleInfo, err := loadModuleInfo(versionsYamlPath)
			if errors.Is(err, ErrModuleRelativePath) {
				modulesChan <- ModuleNeedsReleaseInfo{
					// relative module path not available in that specifically is the error
					Error: err,
				}
			} else if err != nil {
				modulesChan <- ModuleNeedsReleaseInfo{
					ModulePath: versionsYamlPath, // Including the path so that the error can be traced if needed
					Error:      err,
				}
			}
			modulesChan <- getModuleNeedsReleaseInfo(moduleInfo)
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

func getUnreleasedChangesSummary(path, previousReleaseRef string) string {
	switch {
	case previousReleaseRef == autoReleaseRef:
		return "yes, AUTORELEASE pending."
	case previousReleaseRef == InitRef:
		return "Module never had a release. Everything is a change!"
	}

	gitlog, err := getUnreleasedCommitsLog(path, previousReleaseRef)
	if err != nil {
		log.Error("Error running git log", "error", err)
		return fmt.Sprintf("error: Unable to fetch changes since last release (%s)", err)
	}
	return gitlog
}

func getUnreleasedCommitsLog(path, previousReleaseRef string) (string, error) {
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
	gitlog, err := git.LogOneLine(repoRoot, revisionRange, path)
	if err != nil {
		return "", fmt.Errorf("failed to comput gitlog on %s since %s: %w", path, revisionRange, err)
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

func getSortedModulesFoundInPath(path string) ([]string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("unable to convert %s to absolute path, cannot detect modules: %w", path, err)
	}

	versionsFiles, err := FindVersionsYamlFilesInPath(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect modules in %s: %w", path, err)
	}

	sort.Strings(versionsFiles) // We want consistent order of modules found.

	return versionsFiles, nil
}

func loadModuleInfo(versionsYamlPath string) (*moduleInfo, error) {
	moduleAbsolutePath := filepath.Dir(versionsYamlPath)
	repoRoot := viper.GetString("repoRoot")
	moduleRelativePath, err := GetRelativeModulePathFrom(versionsYamlPath, repoRoot)
	if err != nil {
		return nil, err
	}

	versions, err := ReadFromFile(versionsYamlPath)
	if err != nil {
		return &moduleInfo{
			moduleRelativePath: moduleRelativePath,
		}, fmt.Errorf("failed to load module found at %s: %w", versionsYamlPath, err)
	}

	return &moduleInfo{
		moduleAbsolutePath:       moduleAbsolutePath,
		moduleRelativePath:       moduleRelativePath,
		versions:                 versions,
		versionsYamlAbsolutePath: versionsYamlPath,
	}, nil
}

func getModuleNeedsReleaseInfo(moduleData *moduleInfo) ModuleNeedsReleaseInfo {
	latestRelease := getLatestRelease(moduleData.versions.ReleasedVersions)
	latestReleaseTimestamp := &latestRelease.Timestamp
	if latestRelease.CommitID == InitRef {
		latestReleaseTimestamp = nil
	}

	commitLog, err := getUnreleasedCommitsLog(moduleData.moduleAbsolutePath, latestRelease.CommitID)
	commitCount := strings.Count(commitLog, "\n")
	var infoErrs error
	if err != nil {
		infoErrs = errors.Join(infoErrs, fmt.Errorf("unable to generate unreleased commit count for module %s %w", moduleData.versions.ID, err))
		commitCount = -1
	}
	// TODO get unreleased Dependency changes (loop over dependency paths...)

	return ModuleNeedsReleaseInfo{
		ModuleID:               moduleData.versions.ID,
		ModulePath:             moduleData.moduleRelativePath,
		LatestReleaseTimestamp: latestReleaseTimestamp,
		UnreleasedCommitsCount: commitCount,
		AutoreleasePending:     hasPendingAutorelease(moduleData.versions.ReleasedVersions),
		Error:                  infoErrs,
	}
}

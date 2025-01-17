package modules

import (
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

// ModuleNeedsReleaseInfo info about a modules latest release and unreleased changes
type ModuleNeedsReleaseInfo struct {
	ModuleID               string     `json:"moduleId"`
	ModulePath             string     `json:"modulePath"`
	LatestReleaseTimestamp *time.Time `json:"latestReleaseTimestamp"` // note: nill if no releases
	UnreleasedCommitsCount int        `json:"unreleasedCommitsCount"`
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

// PrintModuleInfo outputs info about a module at the given relative path
// in pretty format, if said module exists.
// TODO update for more code reuse and info parity with GetNeedsReleaseInfoIn()
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

// GetNeedsReleaseInfoIn prints infor for each module needing release at the given path.
// will return on the first error encountered.
func GetNeedsReleaseInfoIn(path string) ([]*ModuleNeedsReleaseInfo, error) {
	modules, err := loadModulesFoundInPath(path)
	if err != nil {
		return nil, err
	}

	infos := []*ModuleNeedsReleaseInfo{}
	for _, module := range modules {
		needsReleaseInfo := getModuleNeedsReleaseInfo(module)
		infos = append(infos, &needsReleaseInfo)
	}
	return infos, nil
}

func getLatestRelease(releasedVersions []*VersionMetadata) *VersionMetadata {
	lastEntry := releasedVersions[len(releasedVersions)-1]
	if lastEntry.CommitID == autoReleaseRef && len(releasedVersions) > 1 {
		// Return the one before last if the last is a pending autorelease
		return releasedVersions[len(releasedVersions)-2]
	}
	return lastEntry
}

func getUnreleasedChangesSummary(path, previousReleaseRef string) string {
	switch {
	case previousReleaseRef == autoReleaseRef:
		return "yes, AUTORELEASE pending."
	case previousReleaseRef == InitRef:
		return "Module never had a release. Everything is a change!"
	case len(previousReleaseRef) != hashLength:
		log.Error("Invalid previous release ref", "previousReleaseRef", previousReleaseRef)
		return "error: Invalid previous release ref, unable to comput unreleased changes."
	}

	repoRoot := viper.GetString("repoRoot")
	revisionRange := fmt.Sprintf("%s..HEAD", previousReleaseRef)
	gitlog, err := git.LogOneLine(repoRoot, revisionRange, path)
	if err != nil {
		log.Error("Error running git log", "error", err)
		return fmt.Sprintf("error: Unable to fetch changes since last release (%s)", revisionRange)
	}
	// We could also list files changed and not only commits here.
	return gitlog
}

func getUnreleasedCommitsCount(path, previousReleaseRef string) (int, error) {
	switch {
	case previousReleaseRef == autoReleaseRef:
		return 0, fmt.Errorf("autorelease pending") // TODO still return count but for the release before the autorelease?
	case previousReleaseRef == InitRef:
		return 0, nil
	case len(previousReleaseRef) != hashLength:
		return 0, fmt.Errorf("invalid previous release ref %s", previousReleaseRef)
	}

	repoRoot := viper.GetString("repoRoot")
	revisionRange := fmt.Sprintf("%s..HEAD", previousReleaseRef)
	// TODO can we move this to a struct/interface? that would make mocking possible and testing easier
	// However likely still need a mock kaeter module for other parts of this so it's only relatively easier...
	gitlog, err := git.LogOneLine(repoRoot, revisionRange, path)
	if err != nil {
		return 0, fmt.Errorf("failed to comput gitlog on %s: %w", path, err)
	}

	return strings.Count(gitlog, "\n"), nil
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

func loadModulesFoundInPath(path string) ([]*moduleInfo, error) {
	allFoundModules := []*moduleInfo{}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("unable to convert %s to absolute path, cannot detect modules: %w", path, err)
	}

	versionsFiles, err := FindVersionsYamlFilesInPath(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect modules in %s: %w", path, err)
	}

	for _, versionsYamlPath := range versionsFiles {
		versions, err := ReadFromFile(versionsYamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load module found at %s: %w", versionsYamlPath, err)
		}

		moduleAbsolutePath := filepath.Dir(versionsYamlPath)
		repoRoot := viper.GetString("repoRoot")
		moduleRelativePath, err := filepath.Rel(repoRoot, moduleAbsolutePath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine module relative path for %s: %w", moduleAbsolutePath, err)
		}

		allFoundModules = append(allFoundModules, &moduleInfo{
			moduleAbsolutePath:       moduleAbsolutePath,
			moduleRelativePath:       moduleRelativePath,
			versions:                 versions,
			versionsYamlAbsolutePath: versionsYamlPath,
		})
	}

	return allFoundModules, nil
}

func getModuleNeedsReleaseInfo(moduleData *moduleInfo) ModuleNeedsReleaseInfo {
	latestRelease := getLatestRelease(moduleData.versions.ReleasedVersions)
	latestReleaseTimestamp := &latestRelease.Timestamp
	if latestRelease.CommitID == InitRef {
		latestReleaseTimestamp = nil
	}
	// TODO if autorelease is pending do we want to look 1 more release behind?

	commitCount, err := getUnreleasedCommitsCount(moduleData.moduleAbsolutePath, latestRelease.CommitID)
	if err != nil {
		log.Error("unable to generate unreleased commit count for module", "moduleID", moduleData.versions.ID, "error", err)
		commitCount = -1
	}
	// TODO get unreleased Dependency changes (loop over dependency paths...)

	return ModuleNeedsReleaseInfo{
		ModuleID:               moduleData.versions.ID,
		ModulePath:             moduleData.moduleRelativePath,
		LatestReleaseTimestamp: latestReleaseTimestamp,
		UnreleasedCommitsCount: commitCount,
	}
}

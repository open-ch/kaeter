package modules

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/open-ch/kaeter/kaeter/git"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
)

// Style definitions.
var (
	subtle      = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
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
func PrintModuleInfo(path string) {
	versions, err := loadModule(path)
	if err != nil {
		_, _ = fmt.Println(errorBox.Render(lipgloss.JoinVertical(lipgloss.Left,
			"Unable to load module at", path,
			"because", err.Error(),
		)))
		return
	}

	lastRelease := versions.ReleasedVersions[len(versions.ReleasedVersions)-1]
	latestRelease := getLatestRelease(versions.ReleasedVersions)
	estReleaseAge := "∞"
	if latestRelease.CommitID != "INIT" {
		estReleaseAge = fmt.Sprintf("%.f", time.Since(latestRelease.Timestamp).Hours()/24)
	}
	unreleasedChanges := getUnreleasedChanges(path, latestRelease.CommitID)

	_, _ = fmt.Println(moduleBox.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			moduleHeader.Render("Module ID: ", versions.ID),
			fmt.Sprintf("Path: %s", path),
			fmt.Sprintf("Releases: %d", len(versions.ReleasedVersions)-1), // Ignore the 0.0.0 INIT version
			fmt.Sprintf("Previous release: %s", latestRelease.Number),
			fmt.Sprintf("Days since then: %s", estReleaseAge),
			fmt.Sprintf("Last version: %s", lastRelease.Number),
			fmt.Sprintf("Unreleased changes:\n%s", unreleasedChanges),
		),
	))
}

func getLatestRelease(releasedVersions []*kaeter.VersionMetadata) *kaeter.VersionMetadata {
	lastEntry := releasedVersions[len(releasedVersions)-1]
	if lastEntry.CommitID != "AUTORELEASE" && len(releasedVersions) > 1 {
		// Return the one before last if the last is a pending autorelease
		return releasedVersions[len(releasedVersions)-2]
	}
	return lastEntry
}

func getUnreleasedChanges(path, previousReleaseRef string) string {
	if previousReleaseRef == "AUTORELEASE" {
		return "yes, AUTORELEASE pending."
	}
	if previousReleaseRef == "INIT" {
		return "Module never had a release. Everything is a change!"
	}

	// TODO validate that previousReleaseRef is a hash and not something else
	revisionRange := fmt.Sprintf("%s..HEAD", previousReleaseRef)
	log, err := git.Log(path, "--oneline", revisionRange, path)
	if err != nil {
		return fmt.Sprintf("error: Failed log changes since last release (%s): %v", revisionRange, err)
	}
	// We could also list files changed and not only commits here.
	return log
}

func loadModule(path string) (*kaeter.Versions, error) {
	absVersionsPath, err := kaeter.GetVersionsFilePath(path)
	if err != nil {
		return nil, err
	}

	versions, err := kaeter.ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

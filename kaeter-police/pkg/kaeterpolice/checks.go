package kaeterpolice

import (
	"fmt"
	"os"
	"path/filepath"

	kaeter "github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/open-ch/go-libs/fsutils"
)

const readmeFile = "README.md"
const changelogFile = "CHANGELOG.md"

// CheckModulesStartingFrom recursively looks for modules (has versions.yaml) and validates
// they have the required files.
func CheckModulesStartingFrom(path string) error {
	root, err := fsutils.SearchClosestParentContaining(path, ".git")
	if err != nil {
		return err
	}

	allVersionsFiles, err := fsutils.SearchByFileNameRegex(root, kaeter.VersionsFileNameRegex)
	if err != nil {
		return err
	}

	for _, absVersionFilePath := range allVersionsFiles {
		absModulePath := filepath.Dir(absVersionFilePath)
		if err := checkExistence(readmeFile, absModulePath); err != nil {
			return fmt.Errorf("README existence check failed: %s", err.Error())
		}

		if err := checkExistence(changelogFile, absModulePath); err != nil {
			return fmt.Errorf("CHANGELOG existence check failed: %s", err.Error())
		}

		err = checkChangelog(absVersionFilePath, filepath.Join(absModulePath, changelogFile))
		if err != nil {
			return fmt.Errorf("CHANGELOG version check failed: %s", err.Error())
		}
	}

	return nil
}

func checkExistence(file string, absModulePath string) error {
	info, err := os.Stat(absModulePath)
	if err != nil {
		return fmt.Errorf("Error in getting FileInfo about '%s': %s", absModulePath, err.Error())
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", info.Name())
	}
	absFilePath := filepath.Join(absModulePath, file)

	_, err = os.Stat(absFilePath)
	if err != nil {
		return fmt.Errorf("Error in getting FileInfo about '%s': %s", file, err.Error())
	}

	return nil
}

func checkChangelog(absVersionsPath string, absChangelogPath string) error {
	versions, err := kaeter.ReadFromFile(absVersionsPath)
	if err != nil {
		return err
	}

	changelog, err := ReadFromFile(absChangelogPath)
	if err != nil {
		return fmt.Errorf("Error in parsing %s: %s", absVersionsPath, err.Error())
	}

	changelogVersions := make(map[string]bool)
	for _, entry := range changelog.Entries {
		changelogVersions[entry.Version.String()] = true
	}

	for _, releasedVersion := range versions.ReleasedVersions {
		// the typical INIT release looks like "0.0.0: 1970-01-01T00:00:00Z|INIT", and it is often not report in the changelog
		if releasedVersion.CommitID != "INIT" {
			if _, exists := changelogVersions[releasedVersion.Number.String()]; !exists {
				return fmt.Errorf("Version %s does not exists in '%s'", releasedVersion.Number.String(), absChangelogPath)
			}
		}
	}

	return nil
}

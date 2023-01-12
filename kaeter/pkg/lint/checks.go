package lint

import (
	"fmt"
	"os"
	"path/filepath"

	kaeter "github.com/open-ch/kaeter/kaeter/pkg/kaeter"

	"github.com/open-ch/go-libs/fsutils"
)

const readmeFile = "README.md"
const changelogMDFile = "CHANGELOG.md"
const changelogCHANGESFile = "CHANGES"

// CheckModulesStartingFrom finds the root of the git repo
// then recursively looks for modules (having versions.yaml) and
// validates they have the required files.
// Returns on the first error encountered.
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
		if err := CheckModuleFromVersionsFile(absVersionFilePath); err != nil {
			return err
		}
	}

	return nil
}

// CheckModuleFromVersionsFile validates the kaeter module
// from a versions.yaml file checking that the required
// files are present.
func CheckModuleFromVersionsFile(versionsPath string) error {
	versions, err := kaeter.ReadFromFile(versionsPath)
	if err != nil {
		return fmt.Errorf("versions.yaml parsing failed: %s", err.Error())
	}

	absModulePath := filepath.Dir(versionsPath)
	if err := checkExistence(readmeFile, absModulePath); err != nil {
		return fmt.Errorf("README existence check failed: %s", err.Error())
	}

	noCHANGESerr := checkExistence(changelogCHANGESFile, absModulePath)
	if noCHANGESerr == nil {
		err := validateCHANGESFile(filepath.Join(absModulePath, changelogCHANGESFile), versions)
		if err != nil {
			return fmt.Errorf("CHANGES versions check failed: %s", err.Error())
		}
		return nil
	}

	noChangelogMDerr := checkExistence(changelogMDFile, absModulePath)
	if noChangelogMDerr == nil {
		err := checkChangelog(filepath.Join(absModulePath, changelogMDFile), versions)
		if err != nil {
			return fmt.Errorf("CHANGELOG versions check failed: %s", err.Error())
		}
		return nil
	}

	specFile, noSpecErr := findSpecFile(absModulePath)
	if noSpecErr == nil {
		err := checkSpecChangelog(filepath.Join(absModulePath, specFile), versions)
		if err != nil {
			return fmt.Errorf("spec versions check failed: %s", err.Error())
		}
		return nil
	}

	return fmt.Errorf(
		"CHANGELOG existence check failed: a %s, %s or .spec file is required for the module at %s",
		changelogMDFile, changelogCHANGESFile, absModulePath,
	)
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

func checkChangelog(changelogPath string, versions *kaeter.Versions) error {
	changelog, err := ReadFromFile(changelogPath)
	if err != nil {
		return fmt.Errorf("Error in parsing %s: %s", changelogPath, err.Error())
	}

	changelogVersions := make(map[string]bool)
	for _, entry := range changelog.Entries {
		changelogVersions[entry.Version.String()] = true
	}

	for _, releasedVersion := range versions.ReleasedVersions {
		if releasedVersion.CommitID == "INIT" {
			continue // Ignore Kaeter's default INIT releases ("0.0.0: 1970-01-01T00:00:00Z|INIT")
		}
		if _, exists := changelogVersions[releasedVersion.Number.String()]; !exists {
			return fmt.Errorf("Version %s does not exists in '%s'", releasedVersion.Number.String(), changelogPath)
		}
	}

	return nil
}

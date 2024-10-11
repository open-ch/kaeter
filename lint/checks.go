package lint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/modules"
)

const readmeFile = "README.md"
const changelogMDFile = "CHANGELOG.md"
const changelogCHANGESFile = "CHANGES"

// CheckModulesStartingFrom finds the root of the git repo
// then recursively looks for modules (having versions.yaml) and
// validates they have the required files.
// Returns on the first error encountered.
func CheckModulesStartingFrom(path string) error {
	root, err := git.ShowTopLevel(path)
	if err != nil {
		return err
	}

	allVersionsFiles, err := modules.FindVersionsYamlFilesInPath(root)
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
	var allErrors error
	absModulePath := filepath.Dir(versionsPath)
	versions, err := modules.ReadFromFile(versionsPath)
	if err != nil {
		versions = &modules.Versions{}
		allErrors = errors.Join(allErrors, fmt.Errorf("versions.yaml parsing failed: %s", err.Error()))
	}

	err = checkforValidREADME(absModulePath)
	allErrors = errors.Join(allErrors, err)

	err = checkForValidChangelog(versions, absModulePath)
	allErrors = errors.Join(allErrors, err)

	return allErrors
}

func checkforValidREADME(absModulePath string) error {
	if err := checkExistence(readmeFile, absModulePath); err != nil {
		return fmt.Errorf("existence check failed for README: %s", err.Error())
	}
	return nil
}

func checkForValidChangelog(versions *modules.Versions, absModulePath string) error {
	noCHANGESerr := checkExistence(changelogCHANGESFile, absModulePath)
	if noCHANGESerr == nil {
		err := validateCHANGESFile(filepath.Join(absModulePath, changelogCHANGESFile), versions)
		if err != nil {
			return fmt.Errorf("versions check failed for CHANGES: %s", err.Error())
		}
		return nil
	}

	noChangelogMDerr := checkExistence(changelogMDFile, absModulePath)
	if noChangelogMDerr == nil {
		err := checkMarkdownChangelog(filepath.Join(absModulePath, changelogMDFile), versions)
		if err != nil {
			return fmt.Errorf("versions check failed for CHANGELOG: %s", err.Error())
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
		"existence check failed for CHANGELOG: a %s, %s or .spec file is required for the module at %s",
		changelogMDFile, changelogCHANGESFile, absModulePath,
	)
}

func checkExistence(file, absModulePath string) error {
	info, err := os.Stat(absModulePath)
	if err != nil {
		return fmt.Errorf("error in getting FileInfo about '%s': %s", absModulePath, err.Error())
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", info.Name())
	}
	absFilePath := filepath.Join(absModulePath, file)

	_, err = os.Stat(absFilePath)
	if err != nil {
		return fmt.Errorf("error in getting FileInfo about '%s': %s", file, err.Error())
	}

	return nil
}

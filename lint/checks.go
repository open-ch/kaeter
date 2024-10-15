package lint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-ch/kaeter/modules"
)

const readmeFile = "README.md"
const changelogMDFile = "CHANGELOG.md"
const changelogCHANGESFile = "CHANGES"

// CheckModulesStartingFrom recursively looks for modules (having versions.yaml) and
// validates they have the required files.
// If modules are successfully detected, returns joined error containing errors
// found on all the detected modules.
func CheckModulesStartingFrom(path string) error {
	allVersionsFiles, moduleErrors := modules.FindVersionsYamlFilesInPath(path)
	if moduleErrors != nil {
		return moduleErrors
	}

	for _, absVersionFilePath := range allVersionsFiles {
		err := CheckModuleFromVersionsFile(path, absVersionFilePath)
		// if err not nil we could look up the module id and include that in the printout
		moduleErrors = errors.Join(moduleErrors, err)
	}
	return moduleErrors
}

// CheckModuleFromVersionsFile validates the kaeter module
// from a versions.yaml file checking that the required
// files are present.
func CheckModuleFromVersionsFile(repoRoot, versionsPath string) error {
	absModulePath := filepath.Dir(versionsPath)
	versions, allErrors := checkForValidVersionsFile(repoRoot, versionsPath)

	err := checkForValidREADME(absModulePath)
	allErrors = errors.Join(allErrors, err)

	err = checkForValidChangelog(versions, absModulePath)
	allErrors = errors.Join(allErrors, err)

	return allErrors
}

func checkForValidVersionsFile(repoRoot, versionsPath string) (*modules.Versions, error) {
	versions, versionErrors := modules.ReadFromFile(versionsPath)
	if versionErrors != nil {
		versions = &modules.Versions{}
		versionErrors = fmt.Errorf("versions.yaml parsing failed: %w", versionErrors)
	}

	for _, moduleDependency := range versions.Dependencies {
		fullPath := filepath.Join(repoRoot, moduleDependency)
		_, err := os.Stat(fullPath)
		if err != nil {
			versionErrors = errors.Join(versionErrors, fmt.Errorf("unable to locate module dependency '%s': %w", moduleDependency, err))
		}
	}

	return versions, versionErrors
}

func checkForValidREADME(absModulePath string) error {
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

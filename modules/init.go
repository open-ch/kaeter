package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type newModuleData struct {
	ID               string
	VersioningScheme string
}

// InitializationConfig holds the parameters that can be tweaked
// when initializing a new kaeter module.
type InitializationConfig struct {
	InitChangelog    bool
	InitReadme       bool
	ModuleID         string
	ModulePath       string
	VersioningScheme string
}

// Initialize initializes a versions.yaml file at the specified path and a module identified with 'moduleId'.
// path should point to the module's directory and the other required files (readme and changelog).
func Initialize(config InitializationConfig) (*Versions, error) {
	sanitizedVersioningScheme, err := validateVersioningScheme(config.VersioningScheme)
	if err != nil {
		return nil, err
	}
	absPath, err := getAbsoluteNewModulePath(config.ModulePath) // TODO change this to insure module path (and create ala mkdir -p)
	if err != nil {
		return nil, err
	}

	versions, err := initVersionsFile(absPath, config.ModuleID, sanitizedVersioningScheme)
	if err != nil {
		return nil, err
	}

	var readmePath string
	if config.InitReadme {
		readmePath, err = initReadmeIfAbsent(absPath)
		if err != nil {
			return nil, err
		}
	}

	if config.InitChangelog {
		err = initChangelogIfAbsent(absPath)
		if err != nil {
			return nil, err
		}
	}

	if config.InitReadme && config.InitChangelog {
		err = appendChangelogLinkToFile(readmePath, "CHANGELOG.md")
		if err != nil {
			return nil, err
		}
	}

	// TODO also initialize makefile based on template

	return versions, nil
}

func validateVersioningScheme(versioningScheme string) (string, error) {
	// TODO clean up we have consts SemVer/CalVer/AnyStringVer with lower case strings and the actual CamelCase we want as plain strings
	// we can use the consts all around by making them CamelCase
	switch strings.ToLower(versioningScheme) {
	case SemVer:
		return "SemVer", nil
	case CalVer:
		return "CalVer", nil
	case AnyStringVer:
		return "AnyStringVer", nil
	}
	return "", fmt.Errorf("unknown versioning scheme: %s", versioningScheme)
}

func getAbsoluteNewModulePath(modulePath string) (string, error) {
	absPath, err := filepath.Abs(modulePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	// TODO replace this with folder creation if it doesn't exist -> os.MkdirAll
	if !info.IsDir() {
		return "", fmt.Errorf("requires a path to an existing directory. %s resolved to %s which is not a directory", modulePath, absPath)
	}
	return absPath, nil
}

func initVersionsFile(moduleAbsPath, moduleID, sanitizedVersioningScheme string) (*Versions, error) {
	versionsPathYaml := filepath.Join(moduleAbsPath, "versions.yaml")
	if _, err := os.Stat(versionsPathYaml); !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot init a module with a pre-existing versions.yaml file: %s", versionsPathYaml)
	}
	versionsPathYml := filepath.Join(moduleAbsPath, "versions.yml")
	if _, err := os.Stat(versionsPathYml); !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot init a module with a pre-existing versions.yml file: %s", versionsPathYml)
	}

	tmpl, err := template.New("versions template").Parse(versionsTemplate)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(versionsPathYaml)
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(file, newModuleData{moduleID, sanitizedVersioningScheme})
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return ReadFromFile(versionsPathYaml)
}

func initReadmeIfAbsent(moduleAbsPath string) (string, error) {
	readmePath := filepath.Join(moduleAbsPath, "README.md")
	_, err := os.Stat(readmePath)
	if !os.IsNotExist(err) {
		// File exists, stop here
		return readmePath, nil
	}

	newReadme, e := os.Create(readmePath)
	if e != nil {
		return "", e
	}

	_, err = newReadme.WriteString("BLESS-THE-MEANING-UPON-ME\n")
	if err != nil {
		return "", err
	}
	err = newReadme.Close()
	if err != nil {
		return "", err
	}
	return readmePath, nil
}

func initChangelogIfAbsent(moduleAbsPath string) error {
	changelogPath := filepath.Join(moduleAbsPath, "CHANGELOG.md")
	_, err := os.Stat(changelogPath)
	if !os.IsNotExist(err) {
		// File exists, stop here
		return nil
	}

	newChangelog, err := os.Create(changelogPath)
	if err != nil {
		return err
	}

	_, err = newChangelog.WriteString("# CHANGELOG\n")
	if err != nil {
		return err
	}

	err = newChangelog.Close()
	if err != nil {
		return err
	}

	return nil
}

func appendChangelogLinkToFile(targetPath, relativeChangelogLocation string) error {
	_, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		// File does not exist, stop here
		return err
	}
	targetFile, err := os.OpenFile(targetPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(targetFile, changeLogLink, relativeChangelogLocation)
	if err != nil {
		return err
	}

	err = targetFile.Close()
	if err != nil {
		return err
	}
	return nil
}

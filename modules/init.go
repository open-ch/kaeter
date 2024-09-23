package modules

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/open-ch/kaeter/log"
)

//go:embed versions.tpl.yaml
var versionsTemplate string

//go:embed CHANGELOG.md.tpl
var changelogTemplate string

//go:embed README.md.tpl
var readmeTemplate string

// InitializationConfig holds the parameters that can be tweaked
// when initializing a new kaeter module.
type InitializationConfig struct {
	InitChangelog      bool
	InitReadme         bool
	ModuleID           string
	ModulePath         string
	VersioningScheme   string
	moduleAbsolutePath string
}

// Initialize initializes a kaeter modules with the required files based on the config
// typically the versions.yaml, a readme and a changelog.
func Initialize(config InitializationConfig) (*Versions, error) {
	sanitizedVersioningScheme, err := validateVersioningScheme(config.VersioningScheme)
	if err != nil {
		return nil, err
	}
	config.VersioningScheme = sanitizedVersioningScheme

	absPath, err := validateModulePathAndCreateDir(config.ModulePath)
	if err != nil {
		return nil, err
	}
	config.moduleAbsolutePath = absPath

	versions, err := config.initVersionsFile()
	if err != nil {
		return nil, err
	}

	err = config.initReadmeIfNeeded()
	if err != nil {
		return nil, err
	}

	err = config.initChangelogIfAbsent()
	if err != nil {
		return nil, err
	}

	// TODO also initialize makefile based on template

	return versions, nil
}

func validateVersioningScheme(versioningScheme string) (string, error) {
	// Since we're taking the versioning scheme as an argument we compare it in a case insensitive way
	// (with unicode case folding) to allow the flexibility of handling `"SEMVER"` or `"semver"
	// being parsed from cli but still initializing with a consistent `"SemVer"`.
	switch {
	case strings.EqualFold(versioningScheme, SemVer):
		return SemVer, nil
	case strings.EqualFold(versioningScheme, CalVer):
		return CalVer, nil
	case strings.EqualFold(versioningScheme, AnyStringVer):
		return AnyStringVer, nil
	}
	return "", fmt.Errorf("unknown versioning scheme: %s", versioningScheme)
}

func validateModulePathAndCreateDir(modulePath string) (string, error) {
	absPath, err := filepath.Abs(modulePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(absPath, 0755)
			return absPath, err
		}
		return "", err
	}

	if !info.IsDir() {
		return "", fmt.Errorf("requires a path to an existing directory. %s resolved to %s which is not a directory", modulePath, absPath)
	}

	versionsPathYaml := filepath.Join(absPath, "versions.yaml")
	if _, err := os.Stat(versionsPathYaml); !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot init a module with a pre-existing versions.yaml file: %s", versionsPathYaml)
	}
	versionsPathYml := filepath.Join(absPath, "versions.yml")
	if _, err := os.Stat(versionsPathYml); !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot init a module with a pre-existing versions.yml file: %s", versionsPathYml)
	}

	return absPath, nil
}

func (config *InitializationConfig) initVersionsFile() (*Versions, error) {
	versionsPathYaml := filepath.Join(config.moduleAbsolutePath, "versions.yaml")
	err := config.renderTemplateIfAbsent(versionsTemplate, versionsPathYaml)
	if err != nil {
		return nil, err
	}
	return ReadFromFile(versionsPathYaml)
}

func (config *InitializationConfig) initReadmeIfNeeded() error {
	if !config.InitReadme {
		log.Debug("Skipping readme file creation")
		return nil
	}
	readmePath := filepath.Join(config.moduleAbsolutePath, "README.md")
	return config.renderTemplateIfAbsent(readmeTemplate, readmePath)
}

func (config *InitializationConfig) initChangelogIfAbsent() error {
	if !config.InitChangelog {
		log.Debug("Skipping changelog file creation")
		return nil
	}
	changelogPath := filepath.Join(config.moduleAbsolutePath, "CHANGELOG.md")
	return config.renderTemplateIfAbsent(changelogTemplate, changelogPath)
}

func (config *InitializationConfig) renderTemplateIfAbsent(rawTemplate, renderPath string) error {
	if fileExists(renderPath) {
		return nil
	}

	tmpl, err := template.New(renderPath).Parse(rawTemplate)
	if err != nil {
		return err
	}
	newReadme, e := os.Create(renderPath)
	if e != nil {
		return e
	}

	err = tmpl.Execute(newReadme, config)
	if err != nil {
		_ = newReadme.Close()
		return err
	}
	err = newReadme.Close()
	if err != nil {
		return err
	}
	return nil
}

func fileExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return !os.IsNotExist(err)
}

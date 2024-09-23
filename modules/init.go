package modules

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/open-ch/kaeter/log"

	"github.com/spf13/viper"
)

type templateID int

const (
	templateIDREADME templateID = iota
	templateIDCHANGELOG
	templateIDVersions
)

//go:embed versions.tpl.yaml
var rawTemplateDefaultVersions string

//go:embed CHANGELOG.md.tpl
var rawTemplateDefaultCHANGELOG string

//go:embed README.md.tpl
var rawTemplateDefaultREADME string

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
	err := config.renderTemplateIfAbsent(templateIDVersions, versionsPathYaml)
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
	return config.renderTemplateIfAbsent(templateIDREADME, readmePath)
}

func (config *InitializationConfig) initChangelogIfAbsent() error {
	if !config.InitChangelog {
		log.Debug("Skipping changelog file creation")
		return nil
	}
	changelogPath := filepath.Join(config.moduleAbsolutePath, "CHANGELOG.md")
	return config.renderTemplateIfAbsent(templateIDCHANGELOG, changelogPath)
}

func (config *InitializationConfig) renderTemplateIfAbsent(id templateID, renderPath string) error {
	if fileExists(renderPath) {
		return nil
	}

	tmpl, err := loadTemplate(id)
	if err != nil {
		return err
	}
	file, err := os.Create(renderPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = tmpl.Execute(file, config)
	if err != nil {
		return err
	}
	return nil
}

func loadTemplate(id templateID) (*template.Template, error) {
	switch id {
	case templateIDCHANGELOG:
		if viper.IsSet("templates.default.changelog") {
			return loadExternalTemplate("templates.default.changelog", "default_changelog")
		}
		return template.New("built-in_changelog").Parse(rawTemplateDefaultCHANGELOG)
	case templateIDREADME:
		if viper.IsSet("templates.default.readme") {
			return loadExternalTemplate("templates.default.readme", "default_readme")
		}
		return template.New("built-in_readme").Parse(rawTemplateDefaultREADME)
	case templateIDVersions:
		if viper.IsSet("templates.default.versions") {
			return loadExternalTemplate("templates.default.versions", "default_versions")
		}
		return template.New("built-in_versions").Parse(rawTemplateDefaultVersions)
	default:
		return nil, fmt.Errorf("unknown template type %d", id)
	}
}

func loadExternalTemplate(viperKey, templateName string) (*template.Template, error) {
	rawTemplate, err := os.ReadFile(viper.GetString(viperKey))
	if err != nil {
		return nil, fmt.Errorf("unable to load template from config %s: %w", viperKey, err)
	}
	return template.New(templateName).Parse(string(rawTemplate))
}

func fileExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return !os.IsNotExist(err)
}

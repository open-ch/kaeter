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

const (
	templateTypeREADME    = "readme"
	templateTypeCHANGELOG = "changelog"
	templateTypeVersions  = "versions"
	templateTypeMakefile  = "makefile"
	defaultFlavor         = "default"
)

//go:embed versions.yaml.tpl
var rawTemplateDefaultVersions string

//go:embed CHANGELOG.md.tpl
var rawTemplateDefaultCHANGELOG string

//go:embed README.md.tpl
var rawTemplateDefaultREADME string

//go:embed Makefile.kaeter.tpl
var rawTemplateDefaultMakefile string

// InitializationConfig holds the parameters that can be tweaked
// when initializing a new kaeter module.
type InitializationConfig struct {
	InitChangelog      bool
	InitReadme         bool
	InitMakefile       bool
	ModuleDir          string
	ModuleID           string
	ModulePath         string
	VersioningScheme   string
	Flavor             string
	moduleAbsolutePath string
}

// Initialize initializes a kaeter modules with the required files based on the config
// typically the versions.yaml, a readme and a changelog.
func Initialize(config *InitializationConfig) (*Versions, error) {
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
	config.ModuleDir = filepath.Base(absPath)

	if config.Flavor == "" {
		config.Flavor = defaultFlavor
	}

	versions, err := config.initVersionsFile()
	if err != nil {
		return nil, err
	}

	err = config.initReadmeIfNeeded()
	if err != nil {
		return nil, err
	}

	err = config.initChangelogIfNeeded()
	if err != nil {
		return nil, err
	}

	err = config.initMakefileIfNeeded()
	if err != nil {
		return nil, err
	}

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
	err := config.renderTemplateIfAbsent(templateTypeVersions, versionsPathYaml)
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
	return config.renderTemplateIfAbsent(templateTypeREADME, readmePath)
}

func (config *InitializationConfig) initChangelogIfNeeded() error {
	if !config.InitChangelog {
		log.Debug("Skipping changelog file creation")
		return nil
	}
	changelogPath := filepath.Join(config.moduleAbsolutePath, "CHANGELOG.md")
	return config.renderTemplateIfAbsent(templateTypeCHANGELOG, changelogPath)
}

func (config *InitializationConfig) initMakefileIfNeeded() error {
	if !config.InitMakefile {
		log.Debug("Skipping makefile file creation")
		return nil
	}
	makefilePath := filepath.Join(config.moduleAbsolutePath, "Makefile.kaeter")
	return config.renderTemplateIfAbsent(templateTypeMakefile, makefilePath)
}

func (config *InitializationConfig) renderTemplateIfAbsent(templateType, renderPath string) error {
	if fileExists(renderPath) {
		return nil
	}

	tmpl, err := loadTemplate(templateType, config.Flavor)
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

func loadTemplate(templateType, flavor string) (*template.Template, error) {
	if flavor != defaultFlavor && !viper.IsSet(fmt.Sprintf("templates.%s", flavor)) {
		return nil, fmt.Errorf("template flavor not found in config: %s", flavor)
	}

	var defaultRawTemplate string
	switch templateType {
	case templateTypeCHANGELOG:
		defaultRawTemplate = rawTemplateDefaultCHANGELOG
	case templateTypeREADME:
		defaultRawTemplate = rawTemplateDefaultREADME
	case templateTypeVersions:
		defaultRawTemplate = rawTemplateDefaultVersions
	case templateTypeMakefile:
		defaultRawTemplate = rawTemplateDefaultMakefile
	default:
		return nil, fmt.Errorf("unknown template type %s", templateType)
	}

	templateViperKey := fmt.Sprintf("templates.%s.%s", flavor, templateType)
	if viper.IsSet(templateViperKey) {
		rawTemplate, err := os.ReadFile(viper.GetString(templateViperKey))
		if err != nil {
			return nil, fmt.Errorf("unable to load template from config %s: %w", templateViperKey, err)
		}
		return template.New(fmt.Sprintf("%s_%s", flavor, templateType)).Parse(string(rawTemplate))
	}

	if flavor != defaultFlavor {
		return nil, fmt.Errorf("no template defined for %s", templateViperKey)
	}

	return template.New(fmt.Sprintf("built-in_%s", templateType)).Parse(defaultRawTemplate)
}

func fileExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return !os.IsNotExist(err)
}

package change

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

const separator = string(filepath.Separator)

// LabelCharacters is the list of valid character for a Bazel package
const LabelCharacters = "a-zA-Z0-9-_"

// RepoLabelRegex the regex to find labels of the current repository
var RepoLabelRegex = regexp.MustCompile(fmt.Sprintf("//([%s]+/)*[%s]+(:[%s]+){0,1}", LabelCharacters, LabelCharacters, LabelCharacters))

// PackageLabelRegex the regex to find labels of the current package
var PackageLabelRegex = regexp.MustCompile(fmt.Sprintf(":[%s]+", LabelCharacters))

// KaeterChange contains a map of changed Modules by ids
type KaeterChange struct {
	Modules map[string]modules.KaeterModule
}

// KaeterCheck attempts to find all Kaeter modules and infers based on the
// change set which module were altered
func (d *Detector) KaeterCheck(changes *Information) (kc KaeterChange, err error) {
	kc.Modules = make(map[string]modules.KaeterModule)
	allTouchedFiles := append(append(changes.Files.Added, changes.Files.Modified...), changes.Files.Removed...)

	// For each, resolve Bazel or non-Bazel targets
	for i, m := range d.KaeterModules {
		log.Debug("DetectorKaeter: Inspecting Module", "moduleID", m.ModuleID)
		err = d.checkModuleForChanges(&d.KaeterModules[i], &kc, allTouchedFiles)
		if err != nil {
			return kc, fmt.Errorf("error detecting changes for %s: %w", m.ModuleID, err)
		}
	}
	return kc, nil
}

func (d *Detector) checkModuleForChanges(m *modules.KaeterModule, kc *KaeterChange, allTouchedFiles []string) error {
	if m.ModuleType != "Makefile" {
		log.Warn("DetectorKaeter: skipping unsupported non Makefile types", "moduleID", m.ModuleID)
		return nil
	}

	relativeModulePath := m.ModulePath
	if path.IsAbs(relativeModulePath) {
		relativePath, err := filepath.Rel(d.RootPath, relativeModulePath)
		if err != nil {
			return err
		}
		relativeModulePath = relativePath
	}
	if !strings.HasSuffix(relativeModulePath, separator) {
		relativeModulePath += separator
	}

	// We assume 2 kinds of changes as affecting/changing a module itself:
	// - Changes to the files under the module's base path
	// - Changes to matching one of the modules listed dependency paths
	// for speed and efficiency we return early as soon as one change is detected and stop additional checks.
	for _, file := range allTouchedFiles {
		if d.fileIsModuleChange(file, relativeModulePath) {
			log.Debug("DetectorKaeter: module affected by file changes", "file", file)
			kc.Modules[m.ModuleID] = *m
			return nil
		}

		log.Debug("DetectorKaeter: Checking module for dependency changes", "moduleId", m.ModuleID)
		dependencyChangesDetected, err := d.fileIsDependencyChange(file, m.Dependencies)
		if err != nil {
			return err
		}
		if dependencyChangesDetected {
			kc.Modules[m.ModuleID] = *m
			return nil
		}
	}

	return nil
}

func (*Detector) fileIsModuleChange(file, relativeModulePath string) bool {
	return (relativeModulePath == "."+separator && !path.IsAbs(file)) ||
		strings.HasPrefix(file, relativeModulePath)
}

func (d *Detector) fileIsDependencyChange(file string, dependencyPaths []string) (bool, error) {
	for _, dependency := range dependencyPaths {
		fullPath := filepath.Clean(d.RootPath + separator + dependency)
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			return false, fmt.Errorf("unable to get stats for '%s': %w", dependency, err)
		}
		if fileInfo.IsDir() && !strings.HasSuffix(dependency, separator) {
			dependency += separator
		}
		log.Debug("DetectorKaeter: Checking module", "dependency", dependency)
		if strings.HasPrefix(file, dependency) {
			log.Debug("DetectorKaeter: module afected via dependencies file changes", "file", file)
			return true, nil
		}
	}
	return false, nil
}

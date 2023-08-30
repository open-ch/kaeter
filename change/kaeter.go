package change

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-ch/kaeter//log"
	"github.com/open-ch/kaeter//modules"
)

// LabelCharacters is the list of valid character for a Bazel package
var LabelCharacters = "a-zA-Z0-9-_"

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
	for _, m := range d.KaeterModules {
		log.Debugf("DetectorKaeter: Inspecting Module: %s", m.ModuleID)
		err = d.checkModuleForChanges(&m, &kc, allTouchedFiles)
		if err != nil {
			return kc, fmt.Errorf("error detecting changes for %s: %w", m.ModuleID, err)
		}
	}
	return kc, nil
}

func (d *Detector) checkModuleForChanges(m *modules.KaeterModule, kc *KaeterChange, allTouchedFiles []string) error {
	if m.ModuleType != "Makefile" {
		log.Warnf("DetectorKaeter: skipping unsupported non Makefile type module %s", m.ModuleID)
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

	// we assume that any change affecting this folder or its subfolders affects the module
	// We include 2 kinds of file changes as affecting/changing a module itself:
	// - Changes to the files under the module's base path
	// - Changes to matching one of the modules listed dependency paths
	for _, file := range allTouchedFiles {
		if (relativeModulePath == "." && !path.IsAbs(file)) ||
			strings.HasPrefix(file, relativeModulePath) {
			log.Debugf("DetectorKaeter: File '%s' might affect module", file)
			kc.Modules[m.ModuleID] = *m
			// No need to go through the rest of the files, return fast and move to next module
			return nil
		}
		for _, dependency := range m.Dependencies {
			log.Debugf("DetectorKaeter: Dependency %s for Module: %s", dependency, m.ModuleID)
			if strings.HasPrefix(file, dependency) {
				log.Debugf("DetectorKaeter: File '%s' might affect module", file)
				kc.Modules[m.ModuleID] = *m
			}
		}
	}

	return nil
}

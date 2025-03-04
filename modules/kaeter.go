package modules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-ch/kaeter/log"
)

// KaeterModule contains information about a single module
type KaeterModule struct {
	ModuleID     string            `json:"id"`
	ModulePath   string            `json:"path"`
	ModuleType   string            `json:"type"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	AutoRelease  string            `json:"autoRelease,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	// The following are useful at least as context within the modules package to avoid multiple loads of the same information
	// if the turn out useful beyond the package scope we could make them public tho we have to be careful with impact
	// on the JSON output and module detection, ideally it can be streamlined through the use of the inventory.
	versions *Versions
}

type findResult struct {
	module  *KaeterModule
	errPath string
	err     error
}

var (
	// ErrModuleDependencyPath is generated when stats cannot be loaded for the dependency path
	// in a kaeter module. Likely the path does not or no longer exists.
	ErrModuleDependencyPath = fmt.Errorf("modules: Invalid dependency path")

	// ErrModuleRelativePath happens when the relative path of a module cannot be determined
	// for the given repository root.
	ErrModuleRelativePath = fmt.Errorf("modules: unable to compute relative path")

	// ErrModuleDuplicateID happens when a second module is loaded using the same ID as a
	// previously loaded module.
	ErrModuleDuplicateID = fmt.Errorf("modules: ModuleID must be unique")
)

// GetKaeterModules searches the given path and all sub folders for Kaeter modules.
// A Kaeter module is identified by having a versions.yaml file that is parseable by the Kaeter tooling.
func GetKaeterModules(scanStartDir string) (modules []KaeterModule, err error) {
	findingsChan := streamFoundIn(scanStartDir)

	for result := range findingsChan {
		switch {
		case result.err == nil:
			modules = append(modules, *result.module)
		case errors.Is(result.err, ErrModuleSearch):
			return nil, fmt.Errorf("unable to load modules: %w", result.err)
		case errors.Is(result.err, ErrModuleDependencyPath):
			return nil, fmt.Errorf("invalid module found: %w", result.err)
		case errors.Is(result.err, ErrModuleDuplicateID):
			return nil, fmt.Errorf("duplicate IDs found: %w", result.err)
		default:
			// TODO GetKaeterModules is a library function, avoid logging and return errors to caller
			// - take logger as a parameter (rather than using the global logger)
			// - or return the error in a meaningful way instead
			log.Warn(result.err.Error())
		}
	}
	return modules, nil
}

// streamFoundIn will take the results of all versions.yaml files found under the given path
// then attempt to load the module info for each of these. It will emit a result made of
// either the module info if successful or and error if not successful.
// Possible errors:
// - ErrModuleSearch if the search for versions.yaml failed (emitted only once)
// - ErrModuleDuplicateID per module for every module using an already encountered ID
// - ErrModuleDependencyPath per module when dependencies contain invalid paths
// - ErrModuleRelativePath per module if path isn't valid in repo path
func streamFoundIn(scanStartDir string) chan findResult {
	findingsChan := make(chan findResult)
	uniqueIDs := map[string]bool{}

	// TODO continue reimplmenting module detection to use concurency, doing it for finding the files
	//      above already did a 2x speed up.
	//      based on https://github.com/twpayne/find-duplicates
	//      Next: change findVersionsYamlInPathConcurrent to return a channel, and forward it to
	//      run readKaeterModuleInfo concurently as well
	go func() {
		defer close(findingsChan)

		versionsYamlFiles, err := FindVersionsYamlFilesInPath(scanStartDir)
		if err != nil {
			findingsChan <- findResult{err: err}
			return
		}

		for _, versionsYamlPath := range versionsYamlFiles {
			module, err := readKaeterModuleInfo(versionsYamlPath, scanStartDir)

			if err != nil {
				findingsChan <- findResult{err: err, errPath: versionsYamlPath}
			} else if _, alreadyFound := uniqueIDs[module.ModuleID]; alreadyFound {
				findingsChan <- findResult{
					err:     fmt.Errorf("%w but %s was found multiple times", ErrModuleDuplicateID, module.ModuleID),
					errPath: versionsYamlPath,
				}
			} else {
				uniqueIDs[module.ModuleID] = true
				findingsChan <- findResult{
					module: &module,
				}
			}
		}
	}()

	return findingsChan
}

// GetRelativeModulePathFrom takes the absolute path to a versions.yaml file and returns
// a relative path to the module folder based on the repository root.
func GetRelativeModulePathFrom(versionsYamlPath, rootPath string) (relativeModulePath string, err error) {
	moduleAbsolutePath := filepath.Dir(versionsYamlPath)
	moduleRelativePath, err := filepath.Rel(rootPath, moduleAbsolutePath)
	if err != nil {
		err = errors.Join(ErrModuleRelativePath, err)
		return "", fmt.Errorf("failed to determine module relative path for %s in %s: %w", moduleAbsolutePath, rootPath, err)
	}
	return moduleRelativePath, nil
}

// readKaeterModuleInfo parses the versions.yaml file and returns information about the module
func readKaeterModuleInfo(versionsPath, rootPath string) (module KaeterModule, err error) {
	modulePath, err := GetRelativeModulePathFrom(versionsPath, rootPath)
	if err != nil {
		return module, err
	}
	versions, err := ReadFromFile(versionsPath)
	if err != nil {
		return module, fmt.Errorf("could not load %s: %w", modulePath, err)
	}
	if versions.ID == "" {
		return module, fmt.Errorf("module does not have an identifier: %s", modulePath)
	}

	module = KaeterModule{
		ModuleID:   versions.ID,
		ModulePath: modulePath,
		ModuleType: versions.ModuleType,
		versions:   versions,
	}
	if versions.Metadata != nil && len(versions.Metadata.Annotations) > 0 {
		module.Annotations = versions.Metadata.Annotations
	}
	err = module.parseAndValidateDependencies(versions, rootPath)
	if err != nil {
		return module, err
	}
	err = module.parseAutorelease(versions)
	if err != nil {
		return module, err
	}

	return module, nil
}

func (mod *KaeterModule) parseAndValidateDependencies(versionsFile *Versions, rootPath string) error {
	if len(versionsFile.Dependencies) > 0 {
		mod.Dependencies = versionsFile.Dependencies
	}
	var errs error
	for _, dep := range mod.Dependencies {
		fullPath := filepath.Join(rootPath, dep)
		_, err := os.Stat(fullPath)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("%w in %s for '%s'", ErrModuleDependencyPath, mod.ModuleID, dep))
		}
	}
	return errs
}

func (mod *KaeterModule) parseAutorelease(versionsFile *Versions) error {
	autoReleases := make([]*VersionMetadata, 0)
	for _, releaseData := range versionsFile.ReleasedVersions {
		if releaseData.CommitID == "AUTORELEASE" {
			autoReleases = append(autoReleases, releaseData)
		}
	}

	switch as := len(autoReleases); as {
	case 0:
		// No autorelease found, ok.
	case 1:
		mod.AutoRelease = autoReleases[0].Number.String() // #nosec G602
	default:
		return fmt.Errorf("more than 1 autorelease found in %s", mod.ModulePath)
	}

	return nil
}

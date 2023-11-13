package modules

import (
	"fmt"
	"io/fs"
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
}

// GetKaeterModules searches the repo for all Kaeter modules. A Kaeter module is identified by having a
// versions.yaml file that is parseable by the Kaeter tooling.
func GetKaeterModules(gitRoot string) (modules []KaeterModule, err error) {
	versionsYamlFiles, err := findVersionsYamlInPath(gitRoot)
	if err != nil {
		return modules, err
	}

	for _, versionsYamlPath := range versionsYamlFiles {
		module, err := readKaeterModuleInfo(versionsYamlPath, gitRoot)
		if err != nil {
			// TODO GetKaeterModules is a library function, it's called by kaeter itself
			// - take logger as a parameter (rather than using the global logger)
			// - or return the error in a meaning fullway instead
			log.Errorf("Error: %v", err)
			continue
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func findVersionsYamlInPath(basePath string) ([]string, error) {
	possibleVersionsFiles := make([]string, 0)
	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		basename := filepath.Base(path)
		if basename == "versions.yaml" || basename == "versions.yml" {
			possibleVersionsFiles = append(possibleVersionsFiles, path)
		}
		return nil
	})
	return possibleVersionsFiles, err
}

// readKaeterModuleInfo parses the versions.yaml file and returns information about the module
func readKaeterModuleInfo(versionsPath, rootPath string) (module KaeterModule, err error) {
	modulePath, err := filepath.Rel(rootPath, filepath.Dir(versionsPath))
	if err != nil {
		return module, fmt.Errorf("Could find relative path in root (%s): %w", rootPath, err)
	}
	data, err := os.ReadFile(versionsPath)
	if err != nil {
		return module, fmt.Errorf("Could not read %s: %w", versionsPath, err)
	}
	versions, err := UnmarshalVersions(data)
	if err != nil {
		return module, fmt.Errorf("Could not parse %s: %w", versionsPath, err)
	}
	if versions.ID == "" {
		return module, fmt.Errorf("Module does not have an identifier: %s", versionsPath)
	}

	autoReleases := make([]*VersionMetadata, 0)
	for _, releaseData := range versions.ReleasedVersions {
		if releaseData.CommitID == "AUTORELEASE" {
			log.Infof("Autorelease found for path: %s", modulePath)
			autoReleases = append(autoReleases, releaseData)
		}
	}

	module = KaeterModule{
		ModuleID:   versions.ID,
		ModulePath: modulePath,
		ModuleType: versions.ModuleType,
	}

	if versions.Metadata != nil && len(versions.Metadata.Annotations) > 0 {
		module.Annotations = versions.Metadata.Annotations
	}
	if versions.Dependencies != nil && len(versions.Dependencies) > 0 {
		module.Dependencies = versions.Dependencies
	}
	switch as := len(autoReleases); as {
	case 0:
		// No autorelease found, ok.
	case 1:
		module.AutoRelease = autoReleases[0].Number.String() // #nosec G602
	default:
		return module, fmt.Errorf("More than 1 autorelease found in %s", versionsPath)
	}

	return module, nil
}

package modules

import (
	"fmt"
	"io/fs"
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// KaeterModule contains information about a single module
type KaeterModule struct {
	ModuleID    string            `json:"id"`
	ModulePath  string            `json:"path"`
	ModuleType  string            `json:"type"`
	Annotations map[string]string `json:"annotations,omitempty"`
	AutoRelease string            `json:"autoRelease,omitempty"`
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
			logrus.Errorf("kaeter: error for %s, %v", versionsYamlPath, err)
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
func readKaeterModuleInfo(versionsPath string, rootPath string) (module KaeterModule, err error) {
	modulePath, err := filepath.Rel(rootPath, filepath.Dir(versionsPath))
	if err != nil {
		return module, fmt.Errorf("could find relative path in root (%s): %w", rootPath, err)
	}
	data, err := os.ReadFile(versionsPath)
	if err != nil {
		return module, fmt.Errorf("could not read %s: %w", versionsPath, err)
	}
	versions, err := kaeter.UnmarshalVersions(data)
	if err != nil {
		return module, fmt.Errorf("could not parse %s: %w", versionsPath, err)
	}
	if versions.ID == "" {
		return module, fmt.Errorf("module does not have an identifier: %s", versionsPath)
	}

	autoReleases := make([]*kaeter.VersionMetadata, 0)
	for _, releaseData := range versions.ReleasedVersions {
		if releaseData.CommitID == "AUTORELEASE" {
			logrus.Infof("kaeter: autorelease found %s", releaseData)
			autoReleases = append(autoReleases, releaseData)
		}
	}

	if len(autoReleases) > 1 {
		// TODO error here is good but GetKaeterModules above only prints errors
		// so this wont cause a failure!
		return module, fmt.Errorf("more than 1 autorelease found in %s", versions.ID)
	}

	module = KaeterModule{
		ModuleID:   versions.ID,
		ModulePath: modulePath,
		ModuleType: versions.ModuleType,
	}

	if versions.Metadata != nil && len(versions.Metadata.Annotations) > 0 {
		module.Annotations = versions.Metadata.Annotations
	}
	if len(autoReleases) == 1 {
		module.AutoRelease = autoReleases[0].Number.String()
	}

	return module, nil
}

package modules

import (
	"fmt"
	"io/fs"
	"io/ioutil"
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
}

// GetKaeterModules searches the repo for all Kaeter modules. A Kaeter module is identified by having a
// versions.yaml file that is parseable by the Kaeter tooling.
func GetKaeterModules(gitRoot string) (modules []KaeterModule, err error) {
	// Extract the list of potential Kaeter modules by looking for all versions files.
	modulePath := make([]string, 0)
	err = filepath.WalkDir(gitRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		basename := filepath.Base(path)
		if basename == "versions.yaml" || basename == "versions.yml" {
			modulePath = append(modulePath, path)
		}
		return nil
	})

	if err != nil {
		return
	}

	// Try to parse the versions file, the parseable ones are Kaeter modules.
	for _, path := range modulePath {
		module, err := readKaeterModuleInfo(path, gitRoot)
		if err == nil {
			// error is logged by readKaeterModuleInfo, we skip over modules that do not load silently.
			modules = append(modules, module)
		}
	}
	return
}

// readKaeterModuleInfo parses the versions.yaml file and returns information about the module
func readKaeterModuleInfo(versionsPath string, rootPath string) (module KaeterModule, err error) {
	data, err := ioutil.ReadFile(versionsPath)
	if err != nil {
		logrus.Errorf("Kaeter: Could not read %s: %v", versionsPath, err)
		return
	}
	versions, err := kaeter.UnmarshalVersions(data)
	if err != nil {
		logrus.Errorf("Kaeter: Could not parse %s: %v", versionsPath, err)
		return
	}
	if versions.ID == "" {
		return KaeterModule{}, fmt.Errorf("module does not have an identifier: %s", versionsPath)
	}
	modulePath, err := filepath.Rel(rootPath, filepath.Dir(versionsPath))
	if err != nil {
		logrus.Errorf("Kaeter: Could find relative path in root (%s): %v", rootPath, err)
		return
	}
	module = KaeterModule{
		ModuleID:   versions.ID,
		ModulePath: modulePath,
		ModuleType: versions.ModuleType,
	}

	if versions.Metadata != nil && len(versions.Metadata.Annotations) > 0 {
		module.Annotations = versions.Metadata.Annotations
	}

	return
}

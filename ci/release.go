package ci

import (
	"github.com/open-ch/kaeter/actions"
	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// ReleaseConfig contains the data needed to define and
// configure a specific release
type ReleaseConfig struct {
	DryRun     bool
	ModulePath string
}

// ReleaseSingleModule will load then perform the release actions
// on the specified module:
// - build
// - test
// - release (or snapshot if requested)
// using the latest version number by default.
func (rc *ReleaseConfig) ReleaseSingleModule() error {
	log.Infof("Loading module for release: %s", rc.ModulePath)

	absVersionsPath, err := modules.GetVersionsFilePath(rc.ModulePath)
	if err != nil {
		return err
	}
	versions, err := modules.ReadFromFile(absVersionsPath)
	if err != nil {
		return err
	}
	log.Debug("module release", "versions", versions)

	latestVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
	log.Debug("latest version", "version", latestVersion)

	err = actions.RunModuleRelease(&actions.ModuleRelease{
		DryRun:       rc.DryRun,
		SkipCheckout: true,
		ReleaseTarget: actions.ReleaseTarget{
			ModuleID: versions.ID,
			Version:  latestVersion,
		},
		VersionsYAMLPath: absVersionsPath,
		VersionsData:     versions,
	})
	return err
}

package ci

import (
	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
)

// ReleaseConfig contains the data needed to define and
// configure a specific release
type ReleaseConfig struct {
	DryRun     bool
	ModulePath string
	Logger     *logrus.Logger
}

// ReleaseSingleModule will load then perform the release actions
// on the specified module:
// - build
// - test
// - release (or snapshot if requested)
// using the latest version number by default.
func (rc *ReleaseConfig) ReleaseSingleModule() error {
	log := rc.Logger
	log.Infof("Loading module for release: %s", rc.ModulePath)

	absVersionsPath, err := kaeter.GetVersionsFilePath(rc.ModulePath)
	if err != nil {
		return err
	}
	versions, err := kaeter.ReadFromFile(absVersionsPath)
	if err != nil {
		return err
	}
	log.Debugf("module versions %v\n", versions)

	latestVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
	log.Debugf("latest version: %s\n", latestVersion)

	err = kaeter.RunModuleRelease(&kaeter.ModuleRelease{
		DryRun:       rc.DryRun,
		SkipCheckout: true,
		ReleaseTarget: kaeter.ReleaseTarget{
			ModuleID: versions.ID,
			Version:  latestVersion,
		},
		VersionsYAMLPath: absVersionsPath,
		VersionsData:     versions,
		Logger:           rc.Logger,
	})
	return err
}

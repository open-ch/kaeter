package kaeter

import (
	"time"

	"github.com/sirupsen/logrus"
)

// AutoReleaseConfig contains the configuration for
// which releases to prepare
type AutoReleaseConfig struct {
	Logger         *logrus.Logger
	ModulePath     string
	ReleaseVersion string
	RepositoryRef  string
	RepositoryRoot string
}

// AutoReleaseHash holds the constant for the key we use instead of hashes for autorelease.
const AutoReleaseHash = "AUTORELEASE"

// AutoRelease updates the versions.yaml to request an autorelease from CI once merged
func AutoRelease(config *AutoReleaseConfig) error {
	logger := config.Logger
	refTime := time.Now()

	logger.Debugf("Starting release version %s for %s to %s\n", config.ReleaseVersion, config.ModulePath, config.RepositoryRef)

	// TODO validate that ModulePath is ready for ReleaseVersion (check changelog, and ... like kaeter police would)

	versions, err := config.addAutoReleaseVersionEntry(&refTime)
	if err != nil {
		return err
	}
	releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
	logger.Infof("Done with autorelease prep for %s:%s", versions.ID, releaseVersion)

	return nil
}

func (config *AutoReleaseConfig) addAutoReleaseVersionEntry(refTime *time.Time) (*Versions, error) {
	logger := config.Logger
	logger.Infof("Preparing autorelease of module at %s", config.ModulePath)
	absVersionsPath, err := getVersionsFilePath(config.ModulePath)
	if err != nil {
		return nil, err
	}

	versions, err := ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}
	logger.Infof("Module identifier: %s", versions.ID)
	newReleaseMeta, err := versions.AddRelease(refTime, false, false, config.ReleaseVersion, AutoReleaseHash)
	if err != nil {
		return nil, err
	}

	logger.Infof("Autorelease version: %s", newReleaseMeta.Number.String())
	logger.Debugf("Updated versions.yaml at: %s", absVersionsPath)
	versions.SaveToFile(absVersionsPath)

	return versions, nil
}

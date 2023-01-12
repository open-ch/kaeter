package kaeter

import (
	"fmt"
	"time"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"github.com/open-ch/kaeter/kaeter/pkg/lint"
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

	versions, err := config.addAutoReleaseVersionEntry(&refTime)
	if err != nil {
		return err
	}
	releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
	logger.Infof("Done with autorelease prep for %s:%s", versions.ID, releaseVersion)

	err = config.lintKaeterModule()
	if err != nil {
		logger.Errorln("Error detected on module, resetting changes in version.yaml...")
		resetErr := config.resetChanges()
		if resetErr != nil {
			logger.Errorf(
				"Unexpected error resetting change, please remove %s from versions.yaml manually\n%v\n",
				config.ReleaseVersion,
				resetErr,
			)
		}
		return err
	}

	return nil
}

func (config *AutoReleaseConfig) lintKaeterModule() error {
	absVersionsPath, err := kaeter.GetVersionsFilePath(config.ModulePath)
	if err != nil {
		return err
	}
	// TODO instead of computing and reading versions path & file multiple times
	// load once and pass around directly.
	err = lint.CheckModuleFromVersionsFile(absVersionsPath)
	if err != nil {
		return err
	}

	return nil
}

func (config *AutoReleaseConfig) addAutoReleaseVersionEntry(refTime *time.Time) (*kaeter.Versions, error) {
	logger := config.Logger

	// TODO why not combine GetVersionsFilePath and ReadFromFile? do we need both as separate options?
	// Or can we have 2 reads? GetVersionsFilePath is never needed alone.
	absVersionsPath, err := kaeter.GetVersionsFilePath(config.ModulePath)
	if err != nil {
		return nil, err
	}

	versions, err := kaeter.ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Module identifier: %s", versions.ID)
	newReleaseMeta, err := versions.AddRelease(refTime, false, false, config.ReleaseVersion, AutoReleaseHash)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Autorelease version: %s", newReleaseMeta.Number.String())
	logger.Debugf("Updated versions.yaml at: %s", absVersionsPath)
	versions.SaveToFile(absVersionsPath)

	return versions, nil
}

func (config *AutoReleaseConfig) resetChanges() error {
	logger := config.Logger
	absVersionsPath, err := kaeter.GetVersionsFilePath(config.ModulePath)
	if err != nil {
		return fmt.Errorf("unable to find path to version.yaml for reset", err)
	}

	output, err := gitshell.GitCheckout(config.RepositoryRoot, absVersionsPath)
	if err != nil {
		logger.Debugf("Failed reseting versions.yaml, output:%s", output)
		return fmt.Errorf("failed to reset versions.yaml using git", err)
	}
	return nil
}

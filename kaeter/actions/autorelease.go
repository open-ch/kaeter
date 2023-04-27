package actions

import (
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/git"
	"github.com/open-ch/kaeter/kaeter/hooks"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"github.com/open-ch/kaeter/kaeter/lint"
)

// AutoReleaseConfig contains the configuration for
// which releases to prepare
type AutoReleaseConfig struct {
	Logger         *logrus.Logger
	ModulePath     string
	ReleaseVersion string
	RepositoryRef  string
	RepositoryRoot string
	versionsPath   string
	versions       *kaeter.Versions
}

// AutoReleaseHash holds the constant for the key we use instead of hashes for autorelease.
const AutoReleaseHash = "AUTORELEASE"

// AutoRelease updates the versions.yaml to request an autorelease from CI once merged
func AutoRelease(config *AutoReleaseConfig) error {
	logger := config.Logger
	refTime := time.Now()

	err := config.loadVersions()
	if err != nil {
		return err
	}

	if config.ReleaseVersion == "" {
		logger.Debugln("Version not defined, attempting version hook")
		hookVersion, err := config.getReleaseVersionFromHooks()
		if err != nil {
			return err
		}
		logger.Debugf("Using version from autorelease-hook: %s\n", hookVersion)
		config.ReleaseVersion = hookVersion
	}

	logger.Debugf("Starting release version %s for %s to %s\n", config.ReleaseVersion, config.ModulePath, config.RepositoryRef)
	versions, err := config.addAutoReleaseVersionEntry(&refTime)
	if err != nil {
		return err
	}
	releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
	logger.Infof("Done with autorelease prep for %s:%s", versions.ID, releaseVersion)

	err = config.lintKaeterModule()
	if err != nil {
		logger.Errorln("Error detected on module, reverting changes in version.yaml...")
		resetErr := config.restoreVersions()
		if resetErr != nil {
			logger.Errorf(
				"Unexpected error reverting change, please remove %s from versions.yaml manually\n%v\n",
				config.ReleaseVersion,
				resetErr,
			)
		}
		return err
	}

	return nil
}

func (config *AutoReleaseConfig) getReleaseVersionFromHooks() (string, error) {
	if hooks.HasHook("autorelease-version", config.versions) {
		currentVersion := ""
		releasedVersions := len(config.versions.ReleasedVersions)
		if releasedVersions > 0 {
			currentVersion = config.versions.ReleasedVersions[releasedVersions-1].Number.String()
		}
		return hooks.RunHook(
			"autorelease-version", config.versions,
			config.RepositoryRoot,
			[]string{
				config.ModulePath,
				currentVersion,
			},
		)
		// TODO ideally check that the version is a valid version based on the configured versioning scheme
	}
	return "", errors.New(`flag "version" not set: specifying a version to release is required`)
}

func (config *AutoReleaseConfig) lintKaeterModule() error {
	// TODO instead of computing and reading versions file multiple times load once and pass around directly.
	err := lint.CheckModuleFromVersionsFile(config.versionsPath)
	if err != nil {
		return err
	}

	return nil
}

func (config *AutoReleaseConfig) addAutoReleaseVersionEntry(refTime *time.Time) (*kaeter.Versions, error) {
	logger := config.Logger

	logger.Debugf("Module identifier: %s", config.versions.ID)
	newReleaseMeta, err := config.versions.AddRelease(refTime, kaeter.BumpPatch, config.ReleaseVersion, AutoReleaseHash)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Autorelease version: %s", newReleaseMeta.Number.String())
	logger.Debugf("Updated versions.yaml at: %s", config.versionsPath)
	config.versions.SaveToFile(config.versionsPath)

	return config.versions, nil
}

func (config *AutoReleaseConfig) restoreVersions() error {
	logger := config.Logger
	// We want to restore versions.yaml, whether it is staged or unstaged
	output, err := git.Restore(config.RepositoryRoot, "--staged", "--worktree", config.versionsPath)
	if err != nil {
		logger.Debugf("Failed resetting versions.yaml, output:%s", output)
		return fmt.Errorf("failed to reset versions.yaml using git: %w", err)
	}
	return nil
}

func (config *AutoReleaseConfig) loadVersions() error {
	absVersionsPath, err := kaeter.GetVersionsFilePath(config.ModulePath)
	if err != nil {
		return err
	}
	config.versionsPath = absVersionsPath

	versions, err := kaeter.ReadFromFile(absVersionsPath)
	if err != nil {
		return err
	}
	config.versions = versions
	return nil
}

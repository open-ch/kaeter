package actions

import (
	"errors"
	"fmt"
	"time"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/hooks"
	"github.com/open-ch/kaeter/lint"
	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// AutoReleaseConfig contains the configuration for which releases to prepare
type AutoReleaseConfig struct {
	ModulePath     string
	ReleaseVersion string
	RepositoryRef  string
	RepositoryRoot string
	SkipLint       bool
	versionsPath   string
	versions       *modules.Versions
}

// AutoReleaseHash holds the constant for the key we use instead of hashes for autorelease.
const AutoReleaseHash = "AUTORELEASE"

// AutoRelease updates the versions.yaml to request an autorelease from CI once merged
func AutoRelease(config *AutoReleaseConfig) error {
	refTime := time.Now()

	err := config.loadVersions()
	if err != nil {
		return err
	}

	if config.getLastReleaseEntry().CommitID == "AUTORELEASE" {
		err = config.bumpLastReleaseTimestamp(&refTime)
		if err != nil {
			return err
		}
		return config.validateAutoreleaseAndRevertOnError()
	}

	if config.ReleaseVersion == "" {
		log.Debug("Version not defined, attempting version hook")
		var hookVersion string
		hookVersion, verr := config.getReleaseVersionFromHooks()
		if verr != nil {
			return verr
		}
		log.Debug("Using version from autorelease-hook", "version", hookVersion)
		config.ReleaseVersion = hookVersion
	}

	log.Debug("Starting release",
		"version", config.ReleaseVersion,
		"modulePath", config.ModulePath,
		"repositoryRef", config.RepositoryRef)
	versions, err := config.addAutoReleaseVersionEntry(&refTime)
	if err != nil {
		return err
	}
	releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
	log.Info("Done with autorelease prep", "moduleID", versions.ID, "version", releaseVersion)

	return config.validateAutoreleaseAndRevertOnError()
}

func (config *AutoReleaseConfig) getReleaseVersionFromHooks() (string, error) {
	if hooks.HasHook("autorelease-version", config.versions) {
		currentVersion := ""
		currentHash := ""
		releasedVersions := len(config.versions.ReleasedVersions)
		if releasedVersions > 0 {
			currentVersion = config.versions.ReleasedVersions[releasedVersions-1].Number.String()
			currentHash = config.versions.ReleasedVersions[releasedVersions-1].CommitID
		}
		return hooks.RunHook(
			"autorelease-version", config.versions,
			config.RepositoryRoot,
			[]string{
				config.ModulePath,
				currentVersion,
				currentHash,
			},
		)
		// TODO ideally check that the version is a valid version based on the configured versioning scheme
	}
	return "", errors.New(`flag "version" not set: specifying a version to release is required`)
}

func (config *AutoReleaseConfig) lintKaeterModule() error {
	// TODO instead of computing and reading versions file multiple times load once and pass around directly.
	err := lint.CheckModuleFromVersionsFile(config.RepositoryRoot, config.versionsPath)
	if err != nil {
		return err
	}

	return nil
}

func (config *AutoReleaseConfig) addAutoReleaseVersionEntry(refTime *time.Time) (*modules.Versions, error) {
	log.Debug("Module identifier", "moduleID", config.versions.ID)
	newReleaseMeta, err := config.versions.AddRelease(refTime, modules.BumpPatch, config.ReleaseVersion, AutoReleaseHash)
	if err != nil {
		return nil, err
	}

	log.Debug("Updated versions.yaml", "versionsPath", config.versionsPath, "autoreleaseVersion", newReleaseMeta.Number.String())
	err = config.versions.SaveToFile(config.versionsPath)

	return config.versions, err
}

func (config *AutoReleaseConfig) getLastReleaseEntry() *modules.VersionMetadata {
	return config.versions.ReleasedVersions[len(config.versions.ReleasedVersions)-1]
}

func (config *AutoReleaseConfig) bumpLastReleaseTimestamp(refTime *time.Time) error {
	latestVersion := config.getLastReleaseEntry()
	if config.ReleaseVersion != "" && config.ReleaseVersion != latestVersion.Number.String() {
		return fmt.Errorf("cannot autorelease %s an autorelease is still pending for %s", config.ReleaseVersion, latestVersion.Number)
	}
	log.Warn("Latest version is not yet released", "version", latestVersion.Number, "hash", latestVersion.CommitID)
	log.Info("Bumping existing autorelease timestamp", "timestamp", *refTime)
	latestVersion.Timestamp = *refTime
	return config.versions.SaveToFile(config.versionsPath)
}

func (config *AutoReleaseConfig) validateAutoreleaseAndRevertOnError() error {
	if config.SkipLint {
		return nil
	}

	err := config.lintKaeterModule()
	if err != nil {
		log.Error("Error detected on module, reverting changes in version.yaml...")
		resetErr := config.restoreVersions()
		if resetErr != nil {
			log.Error(
				"Unexpected error reverting change, manually edit versions.yaml to remove version",
				"releaseVersion", config.ReleaseVersion,
				"error",
				resetErr,
			)
		}
		return err
	}

	return nil
}

func (config *AutoReleaseConfig) restoreVersions() error {
	output, err := git.RestoreFile(config.RepositoryRoot, config.versionsPath)
	if err != nil {
		log.Debug("git restore failure", "output", output)
		return fmt.Errorf("failed to reset versions.yaml using git: %w", err)
	}
	return nil
}

func (config *AutoReleaseConfig) loadVersions() error {
	absVersionsPath, err := modules.GetVersionsFilePath(config.ModulePath)
	if err != nil {
		return err
	}
	config.versionsPath = absVersionsPath

	versions, err := modules.ReadFromFile(absVersionsPath)
	if err != nil {
		return err
	}
	config.versions = versions
	return nil
}

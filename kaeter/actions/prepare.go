package actions

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/git"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"github.com/open-ch/kaeter/kaeter/lint"
)

// PrepareReleaseConfig contains the configuration for
// which releases to prepare
type PrepareReleaseConfig struct {
	BumpType            kaeter.SemVerBump
	Logger              *logrus.Logger
	ModulePaths         []string
	RepositoryRef       string
	RepositoryRoot      string
	SkipLint            bool
	UserProvidedVersion string
}

// PrepareRelease will generate a release entry in versions.yaml and create a properly formatted
// release commit
func PrepareRelease(config *PrepareReleaseConfig) error {
	logger := config.Logger
	releaseTargets := make([]kaeter.ReleaseTarget, len(config.ModulePaths))

	refTime := time.Now()
	hash, err := git.ResolveRevision(config.RepositoryRoot, config.RepositoryRef)
	if err != nil {
		return err
	}

	logger.Infof("Release(s) based on %s at ref %s", config.RepositoryRef, hash)

	for i, modulePath := range config.ModulePaths {
		versions, err := config.bumpModule(modulePath, hash, &refTime)
		if err != nil {
			return err
		}
		releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
		releaseTargets[i] = kaeter.ReleaseTarget{ModuleID: versions.ID, Version: releaseVersion}
		logger.Infof("Done preparing release for %s:%s", versions.ID, releaseVersion)

		if config.SkipLint {
			continue
		}

		err = config.lintKaeterModule(modulePath)
		if err != nil {
			logger.Errorln("Error detected on module, reverting changes to version.yaml...")
			resetErr := config.restoreVersions(modulePath)
			if resetErr != nil {
				logger.Errorf(
					"Unexpected error reverting change, please remove %s from versions.yaml manually\n%v\n",
					releaseVersion,
					resetErr,
				)
			}
			return err
		}
	}

	releasePlan := &kaeter.ReleasePlan{Releases: releaseTargets}
	commitMsg, err := releasePlan.ToCommitMessage()
	if err != nil {
		return err
	}

	logger.Debugf("Writing Release Plan to commit with message:\n%s", commitMsg)

	logger.Infof("Committing staged changes...")
	output, err := gitshell.GitCommit(config.RepositoryRoot, commitMsg)
	if err != nil {
		return fmt.Errorf("Failed to commit changes: %s\n%w", output, err)
	}

	logger.Infof("Run 'git log' to check the commit message.")

	return nil
}

func (config *PrepareReleaseConfig) bumpModule(modulePath, releaseHash string, refTime *time.Time) (*kaeter.Versions, error) {
	logger := config.Logger
	logger.Infof("Preparing module: %s", modulePath)
	absVersionsPath, err := kaeter.GetVersionsFilePath(modulePath)
	absModuleDir := filepath.Dir(absVersionsPath)
	if err != nil {
		return nil, err
	}

	versions, err := kaeter.ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Module identifier: %s", versions.ID)
	newReleaseMeta, err := versions.AddRelease(refTime, config.BumpType, config.UserProvidedVersion, releaseHash)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Release version: %s", newReleaseMeta.Number.String())
	logger.Debugf("versions.yaml updated: %s", absVersionsPath)
	versions.SaveToFile(absVersionsPath)

	logger.Debugf("Sating file for commit: %s", absVersionsPath)
	output, err := gitshell.GitAdd(absModuleDir, filepath.Base(absVersionsPath))
	if err != nil {
		return nil, fmt.Errorf("Failed to stage changes: %s\n%w", output, err)
	}

	return versions, nil
}

func (*PrepareReleaseConfig) lintKaeterModule(modulePath string) error {
	absVersionsPath, err := kaeter.GetVersionsFilePath(modulePath)
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

func (config *PrepareReleaseConfig) restoreVersions(modulePath string) error {
	logger := config.Logger
	absVersionsPath, err := kaeter.GetVersionsFilePath(modulePath)
	if err != nil {
		return fmt.Errorf("unable to find path to version.yaml for reset: %w", err)
	}

	// We want to restore versions.yaml, whether it is staged or unstaged
	output, err := git.Restore(config.RepositoryRoot, "--staged", "--worktree", absVersionsPath)
	if err != nil {
		logger.Debugf("Failed reseting versions.yaml, output:%s", output)
		return fmt.Errorf("failed to reset versions.yaml using git: %w", err)
	}
	return nil
}

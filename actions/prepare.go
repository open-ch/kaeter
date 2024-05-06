package actions

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/lint"
	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// PrepareReleaseConfig contains the configuration for
// which releases to prepare
type PrepareReleaseConfig struct {
	BumpType            modules.SemVerBump
	ModulePaths         []string
	RepositoryRef       string
	RepositoryRoot      string
	SkipLint            bool
	UserProvidedVersion string
}

// PrepareRelease will generate a release entry in versions.yaml and create a properly formatted
// release commit
func PrepareRelease(config *PrepareReleaseConfig) error {
	releaseTargets := make([]ReleaseTarget, len(config.ModulePaths))

	refTime := time.Now()
	hash, err := git.ResolveRevision(config.RepositoryRoot, config.RepositoryRef)
	if err != nil {
		return err
	}

	log.Infof("Release(s) based on %s at ref %s", config.RepositoryRef, hash)

	for i, modulePath := range config.ModulePaths {
		var versions *modules.Versions
		versions, err = config.bumpModule(modulePath, hash, &refTime)
		if err != nil {
			return err
		}
		releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
		releaseTargets[i] = ReleaseTarget{ModuleID: versions.ID, Version: releaseVersion}
		log.Infof("Done preparing release for %s:%s", versions.ID, releaseVersion)

		if config.SkipLint {
			continue
		}

		err = config.lintKaeterModule(modulePath)
		if err != nil {
			log.Error("Error detected on module, reverting changes to version.yaml...")
			resetErr := config.restoreVersions(modulePath)
			if resetErr != nil {
				log.Errorf(
					"Unexpected error reverting change, please remove %s from versions.yaml manually\n%v\n",
					releaseVersion,
					resetErr,
				)
			}
			return err
		}
	}

	releasePlan := &ReleasePlan{Releases: releaseTargets}
	commitMsg, err := releasePlan.ToCommitMessage()
	if err != nil {
		return err
	}

	log.Debugf("Writing Release Plan to commit with message:\n%s", commitMsg)

	log.Infof("Committing staged changes...")
	output, err := git.Commit(config.RepositoryRoot, commitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %s\n%w", output, err)
	}

	log.Infof("Run 'git log' to check the commit message.")

	return nil
}

func (config *PrepareReleaseConfig) bumpModule(modulePath, releaseHash string, refTime *time.Time) (*modules.Versions, error) {
	log.Infof("Preparing module: %s", modulePath)
	absVersionsPath, err := modules.GetVersionsFilePath(modulePath)
	absModuleDir := filepath.Dir(absVersionsPath)
	if err != nil {
		return nil, err
	}

	versions, err := modules.ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}
	log.Debugf("Module identifier: %s", versions.ID)
	newReleaseMeta, err := versions.AddRelease(refTime, config.BumpType, config.UserProvidedVersion, releaseHash)
	if err != nil {
		return nil, err
	}

	log.Debugf("Release version: %s", newReleaseMeta.Number.String())
	log.Debugf("versions.yaml updated: %s", absVersionsPath)
	err = versions.SaveToFile(absVersionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed save versions.yaml: %w", err)
	}

	log.Debugf("Sating file for commit: %s", absVersionsPath)
	output, err := git.Add(absModuleDir, filepath.Base(absVersionsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to stage changes: %s\n%w", output, err)
	}

	return versions, nil
}

func (*PrepareReleaseConfig) lintKaeterModule(modulePath string) error {
	absVersionsPath, err := modules.GetVersionsFilePath(modulePath)
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
	absVersionsPath, err := modules.GetVersionsFilePath(modulePath)
	if err != nil {
		return fmt.Errorf("unable to find path to version.yaml for reset: %w", err)
	}

	// We want to restore versions.yaml, whether it is staged or unstaged
	output, err := git.Restore(config.RepositoryRoot, "--staged", "--worktree", absVersionsPath)
	if err != nil {
		log.Debugf("Failed reseting versions.yaml, output:%s", output)
		return fmt.Errorf("failed to reset versions.yaml using git: %w", err)
	}
	return nil
}

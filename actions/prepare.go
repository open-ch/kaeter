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

	refTime := time.Now().UTC()
	hash, err := git.ResolveRevision(config.RepositoryRoot, config.RepositoryRef)
	if err != nil {
		return err
	}

	log.Info("Preparing releases", "repoRef", config.RepositoryRef, "releaseHash", hash)

	for i, modulePath := range config.ModulePaths {
		var versions *modules.Versions
		versions, err = config.bumpModule(modulePath, hash, &refTime)
		if err != nil {
			return err
		}
		releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
		releaseTargets[i] = ReleaseTarget{ModuleID: versions.ID, Version: releaseVersion}
		log.Info("Module prepared", "moduleID", versions.ID, "version", releaseVersion)

		if config.SkipLint {
			continue
		}

		err = config.lintKaeterModule(modulePath)
		if err != nil {
			log.Error("Error detected on module, reverting changes to version.yaml...")
			resetErr := config.restoreVersions(modulePath)
			if resetErr != nil {
				log.Error(
					"Unexpected error reverting change, manually edit versions.yaml to remove version",
					"releaseVersion", releaseVersion,
					"error",
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

	log.Debug("Writing Release Plan to commit", "commitMessage", commitMsg)

	log.Info("Committing staged changes...")
	output, err := git.Commit(config.RepositoryRoot, commitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %s\n%w", output, err)
	}

	log.Info("Run 'git log' to check the commit message.")

	return nil
}

func (config *PrepareReleaseConfig) bumpModule(modulePath, releaseHash string, refTime *time.Time) (*modules.Versions, error) {
	log.Info("Preparing module for bump", "modulePath", modulePath)
	absVersionsPath, err := modules.GetVersionsFilePath(modulePath)
	absModuleDir := filepath.Dir(absVersionsPath)
	if err != nil {
		return nil, err
	}

	versions, err := modules.ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}
	log.Debug("versions file loaded for bump", "moduleId", versions.ID)
	newReleaseMeta, err := versions.AddRelease(refTime, config.BumpType, config.UserProvidedVersion, releaseHash)
	if err != nil {
		return nil, err
	}

	log.Debug("saving new version to file", "newVersion", newReleaseMeta.Number.String(), "versionsYAML", absVersionsPath)
	err = versions.SaveToFile(absVersionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed save versions.yaml: %w", err)
	}

	log.Debug("staging file for commit", "versionsYAML", absVersionsPath)
	output, err := git.Add(absModuleDir, filepath.Base(absVersionsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to stage changes: %s\n%w", output, err)
	}

	return versions, nil
}

func (config *PrepareReleaseConfig) lintKaeterModule(modulePath string) error {
	absVersionsPath, err := modules.GetVersionsFilePath(modulePath)
	if err != nil {
		return err
	}
	// TODO refactor to avoid this which loads the versions.yaml again
	err = lint.CheckModuleFromVersionsFile(lint.CheckConfig{RepoRoot: config.RepositoryRoot}, absVersionsPath)
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

	output, err := git.RestoreFile(config.RepositoryRoot, absVersionsPath)
	if err != nil {
		log.Debug("Failed reseting versions.yaml", "output", output)
		return fmt.Errorf("failed to reset versions.yaml using git: %w", err)
	}
	return nil
}

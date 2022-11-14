package kaeter

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/open-ch/go-libs/fsutils"
	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus"
)

// PrepareReleaseConfig contains the configuration for
// which releases to prepare
type PrepareReleaseConfig struct {
	BumpMajor           bool
	BumpMinor           bool
	Logger              *logrus.Logger
	ModulePaths         []string
	RepositoryRef       string
	RepositoryRoot      string
	UserProvidedVersion string
}

// PrepareRelease will generate a release entry in versions.yaml and create a properly formatted
// release commit
func PrepareRelease(config *PrepareReleaseConfig) error {
	logger := config.Logger
	releaseTargets := make([]ReleaseTarget, len(config.ModulePaths))

	refTime := time.Now()
	hash, err := gitshell.GitResolveRevision(config.RepositoryRoot, config.RepositoryRef)
	if err != nil {
		return err
	}

	logger.Infof("Release based on %s, with commit id %s", config.RepositoryRef, hash)

	for i, modulePath := range config.ModulePaths {
		versions, err := config.bumpModule(modulePath, hash, &refTime)
		if err != nil {
			return err
		}
		releaseVersion := versions.ReleasedVersions[len(versions.ReleasedVersions)-1].Number.String()
		releaseTargets[i] = ReleaseTarget{ModuleID: versions.ID, Version: releaseVersion}
		logger.Infof("Done with release preparations for %s:%s", versions.ID, releaseVersion)
	}

	releasePlan := &ReleasePlan{Releases: releaseTargets}
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

func (config *PrepareReleaseConfig) bumpModule(modulePath, releaseHash string, refTime *time.Time) (*Versions, error) {
	logger := config.Logger
	logger.Infof("Preparing release of module at %s", modulePath)
	absVersionsPath, err := getVersionsFilePath(modulePath)
	absModuleDir := filepath.Dir(absVersionsPath)
	if err != nil {
		return nil, err
	}

	versions, err := ReadFromFile(absVersionsPath)
	if err != nil {
		return nil, err
	}
	logger.Infof("Module has identifier: %s", versions.ID)
	newReleaseMeta, err := versions.AddRelease(refTime, config.BumpMajor, config.BumpMinor, config.UserProvidedVersion, releaseHash)
	if err != nil {
		return nil, err
	}

	logger.Infof("Will prepare a release with version: %s", newReleaseMeta.Number.String())
	logger.Infof("Writing versions.yaml file at: %s", absVersionsPath)
	versions.SaveToFile(absVersionsPath)

	logger.Infof("Adding file to commit: %s", absVersionsPath)
	output, err := gitshell.GitAdd(absModuleDir, filepath.Base(absVersionsPath))
	if err != nil {
		return nil, fmt.Errorf("Failed to stage changes: %s\n%w", output, err)
	}

	return versions, nil
}

// pointToVersionsFile checks if the passed path is a directory, then:
//   - checks if there is a versions.yml or .yaml file, and appends the existing one to the abspath if so
//   - appends 'versions.yaml' to it if there is none.
func getVersionsFilePath(modulePath string) (string, error) {
	absModulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absModulePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		versionsFilesFound, err := fsutils.SearchByFileNameRegex(absModulePath, VersionsFileNameRegex)
		if err != nil {
			return "", err
		}
		if len(versionsFilesFound) == 1 {
			return versionsFilesFound[0], nil
		}

		// Multiple matches? Return the file that is at the specified path, otherwise fail
		if len(versionsFilesFound) > 1 {
			for _, match := range versionsFilesFound {
				if path.Dir(match) == absModulePath {
					return match, nil
				}
			}
			return "", fmt.Errorf("Error multiple versions file in: %s", modulePath)
		}

		return filepath.Join(absModulePath, "versions.yaml"), nil
	}
	return absModulePath, nil
}

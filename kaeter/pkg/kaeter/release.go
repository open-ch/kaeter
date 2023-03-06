package kaeter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/open-ch/go-libs/fsutils"
	"github.com/open-ch/go-libs/gitshell"
	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/git"
)

// ReleaseConfig allows customizing how the kaeter release
// will handle the process
type ReleaseConfig struct {
	RepositoryRoot       string
	RepositoryTrunk      string
	ReleaseCommitMessage string
	DryRun               bool // Replaces !really
	SkipCheckout         bool // Replaces nocheckout
	SkipModules          []string
	Logger               *logrus.Logger
}

type moduleRelease struct {
	releaseConfig    *ReleaseConfig
	releaseTarget    ReleaseTarget
	versionsYAMLPath string
	versionsData     *Versions
	headHash         string
}

// RunReleases attempts to release for the modules listed in the
// commit's release plan for the given repository config
// Note: this will return an error on the first release failure, skipping
// any later releases but not roll back any successful ones.
func RunReleases(releaseConfig *ReleaseConfig) error {
	logger := releaseConfig.Logger
	var commitMessage string

	if releaseConfig.ReleaseCommitMessage == "" {
		logger.Debugln("no commit message passed in, attempting to read from HEAD with git")
		headCommitMessage, err := gitshell.GitCommitMessageFromHash(releaseConfig.RepositoryRoot, "HEAD")
		if err != nil {
			return fmt.Errorf("failed to get commit message for HEAD: %w", err)
		}
		commitMessage = headCommitMessage
	} else {
		logger.Debugln("commit-message flag set not reading commit mesage from git")
		commitMessage = releaseConfig.ReleaseCommitMessage
	}

	headHash, err := gitshell.GitResolveRevision(releaseConfig.RepositoryRoot, "HEAD")
	if err != nil {
		return err
	}
	logger.Infof("Repository HEAD at %s", headHash)

	logger.Infof("Commit message: %s", commitMessage)
	rp, err := ReleasePlanFromCommitMessage(commitMessage)
	if err != nil {
		return err
	}
	logger.Infof("Got release plan for the following targets:\n%s", commitMessage)
	for _, releaseMe := range rp.Releases {
		logger.Infof("\t%s", releaseMe.Marshal())
	}
	allModules, err := fsutils.SearchByFileNameRegex(releaseConfig.RepositoryRoot, VersionsFileNameRegex)
	if err != nil {
		return err
	}

	for _, releaseTarget := range rp.Releases {
		skipReleaseTarget := false
		for _, skipModuleID := range releaseConfig.SkipModules {
			if releaseTarget.ModuleID == skipModuleID {
				skipReleaseTarget = true
				break
			}
		}
		if skipReleaseTarget {
			logger.Infof("Skipping module release: %s", releaseTarget.ModuleID)
			continue
		}

		var versionsYAMLPath = ""
		var versionsData *Versions
		for _, isItMe := range allModules {
			vers, err := ReadFromFile(isItMe)
			if err != nil {
				return fmt.Errorf("something went wrong while walking versions.yaml files in the repo: %s - %s",
					isItMe, err)
			}
			if releaseTarget.ModuleID == vers.ID {
				versionsYAMLPath = isItMe
				versionsData = vers
				break
			}
		}
		if versionsYAMLPath == "" {
			return fmt.Errorf("Could not locate module with id %s in repository living in %s",
				releaseTarget.ModuleID, releaseConfig.RepositoryRoot)
		}
		logger.Infof("Module %s found at %s", releaseTarget.ModuleID, versionsYAMLPath)
		err := runReleaseProcess(&moduleRelease{
			releaseConfig,
			releaseTarget,
			versionsYAMLPath,
			versionsData,
			headHash,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func runReleaseProcess(moduleRelease *moduleRelease) error {
	logger := moduleRelease.releaseConfig.Logger
	versionsData := moduleRelease.versionsData

	if moduleRelease.releaseTarget.ModuleID != versionsData.ID {
		return fmt.Errorf("invalid arguments passed: target id %s is not the same as passed module id:%s",
			moduleRelease.releaseTarget.ModuleID, versionsData.ID)
	}

	// TODO: to support (re)releasing older versions should find moduleRelease.releaseTarget.Version
	// rather than compare it to the latest release.
	latestReleaseVersion := versionsData.ReleasedVersions[len(versionsData.ReleasedVersions)-1]
	if latestReleaseVersion.Number.String() != moduleRelease.releaseTarget.Version {
		return fmt.Errorf("release target %s does not correspond to latest version (%s) found in %s",
			moduleRelease.releaseTarget.Marshal(), latestReleaseVersion.Number.String(), moduleRelease.versionsYAMLPath)
	}
	modulePath := filepath.Dir(moduleRelease.versionsYAMLPath)
	makefileName, err := detectModuleMakefile(modulePath)
	if err != nil {
		return err
	}
	if !moduleRelease.releaseConfig.SkipCheckout {
		releaseCommitHash := latestReleaseVersion.CommitID
		trunkBranch := strings.ReplaceAll(moduleRelease.releaseConfig.RepositoryTrunk, "origin/", "")

		if err := git.ValidateCommitIsOnTrunk(modulePath, trunkBranch, releaseCommitHash); err != nil {
			return fmt.Errorf("Invalid release commit:  %w", err)
		}

		logger.Infof("Checking out commit hash of version %s: %s", latestReleaseVersion.Number, releaseCommitHash)
		output, err := gitshell.GitCheckout(modulePath, releaseCommitHash)
		if err != nil {
			logger.Errorf("Failed to checkout release commit %s:\n%s", releaseCommitHash, output)
			return err
		}
	}
	err = runMakeTarget(modulePath, makefileName, "build", moduleRelease.releaseTarget)
	if err != nil {
		return err
	}
	err = runMakeTarget(modulePath, makefileName, "test", moduleRelease.releaseTarget)
	if err != nil {
		return err
	}
	if moduleRelease.releaseConfig.DryRun {
		logger.Warnf("Dry run mode is enabled: not releasing anything.")
	} else {
		err = runMakeTarget(modulePath, makefileName, "release", moduleRelease.releaseTarget)
		if err != nil {
			return err
		}
	}
	if !moduleRelease.releaseConfig.SkipCheckout {
		output, err := gitshell.GitReset(modulePath, moduleRelease.headHash)
		if err != nil {
			logger.Errorf("Failed to checkout back to head %s:\n%s", moduleRelease.headHash, output)
			return err
		}
		logger.Warnf("Repository HEAD reset to commit(%s) in detached head state", moduleRelease.headHash)
	}
	logger.Infof("Done.")
	return nil
}

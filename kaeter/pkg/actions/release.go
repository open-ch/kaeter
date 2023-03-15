package actions

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/open-ch/kaeter/kaeter/git"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
)

// ReleaseConfig allows customizing how the kaeter release
// will handle the process
type ReleaseConfig struct {
	headHash             string
	RepositoryRoot       string
	RepositoryTrunk      string
	ReleaseCommitMessage string
	DryRun               bool // Replaces !really
	SkipCheckout         bool // Replaces nocheckout
	SkipModules          []string
	Logger               *logrus.Logger
}

// RunReleases attempts to release for the modules listed in the
// commit's release plan for the given repository config
// Note: this will return an error on the first release failure, skipping
// any later releases but not roll back any successful ones.
func RunReleases(releaseConfig *ReleaseConfig) error {
	logger := releaseConfig.Logger

	err := releaseConfig.loadReleaseCommitInfo()
	if err != nil {
		return err
	}
	logger.Infof("Repository HEAD at %s", releaseConfig.headHash)
	logger.Infof("Commit message: %s", releaseConfig.ReleaseCommitMessage)

	rp, err := kaeter.ReleasePlanFromCommitMessage(releaseConfig.ReleaseCommitMessage)
	if err != nil {
		return err
	}
	logger.Infof("Got release plan for the following targets:\n%s", releaseConfig.ReleaseCommitMessage)
	for _, releaseMe := range rp.Releases {
		logger.Infof("\t%s", releaseMe.Marshal())
	}
	allModules, err := kaeter.FindVersionsYamlFilesInPath(releaseConfig.RepositoryRoot)
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
		var versionsData *kaeter.Versions
		for _, isItMe := range allModules {
			vers, err := kaeter.ReadFromFile(isItMe)
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

		err := kaeter.RunModuleRelease(&kaeter.ModuleRelease{
			CheckoutRestoreHash: releaseConfig.headHash,
			DryRun:              releaseConfig.DryRun,
			SkipCheckout:        releaseConfig.SkipCheckout,
			ReleaseTarget:       releaseTarget,
			RepositoryTrunk:     releaseConfig.RepositoryTrunk,
			VersionsYAMLPath:    versionsYAMLPath,
			VersionsData:        versionsData,
			Logger:              releaseConfig.Logger,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (releaseConfig *ReleaseConfig) loadReleaseCommitInfo() error {
	logger := releaseConfig.Logger
	headHash, err := git.ResolveRevision(releaseConfig.RepositoryRoot, "HEAD")
	if err != nil {
		return err
	}
	releaseConfig.headHash = headHash

	if releaseConfig.ReleaseCommitMessage != "" {
		logger.Debugln("commit-message flag set not reading commit mesage from git")
		return nil
	}

	logger.Debugln("no commit message passed in, attempting to read from HEAD with git")
	headCommitMessage, err := git.GetCommitMessageFromRef(releaseConfig.RepositoryRoot, "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get commit message for HEAD: %w", err)
	}
	releaseConfig.ReleaseCommitMessage = headCommitMessage
	return nil
}

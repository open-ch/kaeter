package actions

import (
	"fmt"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/inventory"
	"github.com/open-ch/kaeter/log"
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
}

// RunReleases attempts to release for the modules listed in the
// commit's release plan for the given repository config
// Note: this will return an error on the first release failure, skipping
// any later releases but not roll back any successful ones.
func RunReleases(releaseConfig *ReleaseConfig) error {
	err := releaseConfig.loadReleaseCommitInfo()
	if err != nil {
		return err
	}
	log.Info("Starting release from plan", "RepositoryHEAD", releaseConfig.headHash, "commitMessage", releaseConfig.ReleaseCommitMessage)

	rp, err := ReleasePlanFromCommitMessage(releaseConfig.ReleaseCommitMessage)
	if err != nil {
		return err
	}
	log.Info("Release plan targets loading", "commitMessage", releaseConfig.ReleaseCommitMessage)
	for _, releaseMe := range rp.Releases {
		log.Info("-", "release target", releaseMe.Marshal())
	}
	moduleIventory, err := inventory.InventorizeRepo(releaseConfig.RepositoryRoot)
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
			log.Info("Skipping module release", "moduleID", releaseTarget.ModuleID)
			continue
		}

		targetModule, found := moduleIventory.Lookup[releaseTarget.ModuleID]
		if !found {
			return fmt.Errorf("could not locate module with id %s in repository living in %s",
				releaseTarget.ModuleID, releaseConfig.RepositoryRoot)
		}
		versionsYAMLPath := targetModule.GetVersionsPath()
		log.Info("Module found", "moduleID", releaseTarget.ModuleID, "path", versionsYAMLPath)

		err = RunModuleRelease(&ModuleRelease{
			CheckoutRestoreHash: releaseConfig.headHash,
			DryRun:              releaseConfig.DryRun,
			SkipCheckout:        releaseConfig.SkipCheckout,
			ReleaseTarget:       releaseTarget,
			RepositoryTrunk:     releaseConfig.RepositoryTrunk,
			VersionsYAMLPath:    versionsYAMLPath,
			VersionsData:        targetModule.GetVersions(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (releaseConfig *ReleaseConfig) loadReleaseCommitInfo() error {
	headHash, err := git.ResolveRevision(releaseConfig.RepositoryRoot, "HEAD")
	if err != nil {
		return err
	}
	releaseConfig.headHash = headHash

	if releaseConfig.ReleaseCommitMessage != "" {
		log.Debug("commit-message flag set not reading commit mesage from git")
		return nil
	}

	log.Debug("no commit message passed in, attempting to read from HEAD with git")
	headCommitMessage, err := git.GetCommitMessageFromRef(releaseConfig.RepositoryRoot, "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get commit message for HEAD: %w", err)
	}
	releaseConfig.ReleaseCommitMessage = headCommitMessage
	return nil
}

package actions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/open-ch/kaeter/git"
	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// ModuleRelease allows defining the parameters
// for a single module release.
type ModuleRelease struct {
	CheckoutRestoreHash string
	DryRun              bool
	ReleaseTarget       ReleaseTarget
	RepositoryTrunk     string
	SkipCheckout        bool
	VersionsData        *modules.Versions
	VersionsYAMLPath    string
}

// RunModuleRelease performs a release (possibly dry-run or snapshot)
// based on the ModuleRelease config and handles calling the make targets.
// Note this only supports releasing the latest version from versions.yaml.
func RunModuleRelease(moduleRelease *ModuleRelease) error {
	versionsData := moduleRelease.VersionsData

	if moduleRelease.ReleaseTarget.ModuleID != versionsData.ID {
		return fmt.Errorf("invalid arguments passed: target id %s is not the same as passed module id:%s",
			moduleRelease.ReleaseTarget.ModuleID, versionsData.ID)
	}

	// To support (re)releasing older versions should find moduleRelease.releaseTarget.Version
	// rather than compare it to the latest release.
	latestReleaseVersion := versionsData.ReleasedVersions[len(versionsData.ReleasedVersions)-1]
	if latestReleaseVersion.Number.String() != moduleRelease.ReleaseTarget.Version {
		return fmt.Errorf("release target %s does not correspond to latest version (%s) found in %s",
			moduleRelease.ReleaseTarget.Marshal(), latestReleaseVersion.Number.String(), moduleRelease.VersionsYAMLPath)
	}
	modulePath := filepath.Dir(moduleRelease.VersionsYAMLPath)
	makefileName, err := detectModuleMakefile(modulePath)
	if err != nil {
		return err
	}
	if !moduleRelease.SkipCheckout {
		// TODO ideally refactor the commit selection (latestReleaseVersion above) and ValidateCommitIsOnTrunk outside
		// of the release helper.
		releaseCommitHash := latestReleaseVersion.CommitID
		trunkBranch := strings.ReplaceAll(moduleRelease.RepositoryTrunk, "origin/", "")

		if err := git.ValidateCommitIsOnTrunk(modulePath, trunkBranch, releaseCommitHash); err != nil {
			return fmt.Errorf("Invalid release commit:  %w", err)
		}

		log.Infof("Checking out commit hash of version %s: %s", latestReleaseVersion.Number, releaseCommitHash)
		output, err := git.Checkout(modulePath, releaseCommitHash)
		if err != nil {
			log.Errorf("Failed to checkout release commit %s:\n%s", releaseCommitHash, output)
			return err
		}
	}
	err = runMakeTarget(modulePath, makefileName, "build", moduleRelease.ReleaseTarget)
	if err != nil {
		return err
	}
	err = runMakeTarget(modulePath, makefileName, "test", moduleRelease.ReleaseTarget)
	if err != nil {
		return err
	}
	if moduleRelease.DryRun {
		log.Warnf("Dry run mode is enabled: not releasing anything.")
	} else {
		err = runMakeTarget(modulePath, makefileName, "release", moduleRelease.ReleaseTarget)
		if err != nil {
			return err
		}
	}
	if !moduleRelease.SkipCheckout {
		output, err := git.ResetHard(modulePath, moduleRelease.CheckoutRestoreHash)
		if err != nil {
			log.Errorf("Failed to checkout back to head %s:\n%s", moduleRelease.CheckoutRestoreHash, output)
			return err
		}
		log.Warnf("Repository HEAD reset to commit(%s) in detached head state", moduleRelease.CheckoutRestoreHash)
	}
	log.Infof("Done.")
	return nil
}

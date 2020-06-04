package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"github.com/open-ch/go-libs/fsutils"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	// If this is false, only does a dry run (ie, builds and runs tests but does not produce any release)
	var really bool

	releaseCmd := &cobra.Command{
		Use:   "release",
		Short: "Executes a release plan.",
		Long: `Executes a release plan: currently such a plan can only be provided via the last commit in the repository
on which kaeter is being run. See kaeter's doc for more details.'`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runRelease(really)
			if err != nil {
				logger.Errorf("release failed: %s", err)
				os.Exit(1)
			}
		},
	}

	releaseCmd.Flags().BoolVar(&really, "really", false,
		`If set, and if the module is using SemVer, causes a bump in the minor version of the released module.
By default the build number is incremented.`)

	rootCmd.AddCommand(releaseCmd)
}

func runRelease(really bool) error {
	if !really {
		logger.Warnf("'really' flag is set to false: will run build and tests but no release.")
	}
	logger.Infof("Retrieving release plan from last commit...")
	repo, _, err := openRepoAndWorktree(modulePath)
	if err != nil {
		return err
	}
	// TODO make the ref from which to read the release plan configurable
	headRevision, err := repo.ResolveRevision("HEAD")
	if err != nil {
		return err
	}
	commit, err := repo.CommitObject(*headRevision)
	if err != nil {
		return err
	}

	rp, err := kaeter.ReleasePlanFromCommitMessage(commit.Message)
	if err != nil {
		return err
	}
	logger.Infof("Got release plan for the following targets:")
	for _, releaseMe := range rp.Releases {
		logger.Infof("\t%s", releaseMe.Marshal())
	}

	root, err := fsutils.SearchClosestParentContaining(modulePath, ".git")
	if err != nil {
		return err
	}
	// TODO: locate the relevant versions.yml file
	allModules, err := fsutils.SearchByFileName(root, versionsFile)
	if err != nil {
		return err
	}

	for _, target := range rp.Releases {
		// TODO currently we don't expect more than one target, but the day this changes
		//  we should probably stop looping on allModules.
		var targetPath = ""
		var targetVersions *kaeter.Versions
		for _, isItMe := range allModules {
			vers, err := kaeter.ReadFromFile(isItMe)
			if err != nil {
				return fmt.Errorf("something went wrong while walking versions.yml files in the repo: %s - %s",
					isItMe, err)
			}
			if target.ModuleID == vers.ID {
				targetPath = isItMe
				targetVersions = vers
				break
			}
		}
		if targetPath == "" {
			return fmt.Errorf("Could not locate module with id %s in repository living in %s",
				target.ModuleID, root)
		}
		logger.Infof("Module %s found at %s", target.ModuleID, targetPath)
		err := runReleaseProcess(target, targetPath, targetVersions, really)
		if err != nil {
			return err
		}
	}

	return nil
}

func runReleaseProcess(
	releaseTarget kaeter.ReleaseTarget,
	versionsPath string,
	versionsData *kaeter.Versions,
	really bool) error {

	lastAdded := versionsData.ReleasedVersions[len(versionsData.ReleasedVersions)-1]
	// Should not happen, but if this happens we may as well notify the user...
	if releaseTarget.ModuleID != versionsData.ID {
		return fmt.Errorf("invalid arguments passed: target id %s is not the same as passed module id:%s",
			releaseTarget.ModuleID, versionsData.ID)
	}
	if lastAdded.Number.GetVersionString() != releaseTarget.Version {
		return fmt.Errorf("release target %s does not correspond to latest version (%s) found in %s",
			releaseTarget.Marshal(), lastAdded.Number.GetVersionString(), versionsPath)
	}

	// TODO check we have make commands
	// TODO if we support other tools than make, we need to refactor things
	modulePath := filepath.Dir(versionsPath)
	_, err := checkModuleHasMakefile(modulePath)
	if err != nil {
		return fmt.Errorf("module %s has no Makefile", modulePath)
	}

	// TODO: actually checkout the target commit ID before running the make targets.

	err = runMakeTarget(modulePath, "build", releaseTarget)
	if err != nil {
		return fmt.Errorf("failed to run 'build' target on module %s: %s", modulePath, err)
	}

	err = runMakeTarget(modulePath, "test", releaseTarget)
	if err != nil {
		return fmt.Errorf("failed to run 'test' target on module %s: %s", modulePath, err)
	}

	if really {
		err = runMakeTarget(modulePath, "release", releaseTarget)
		if err != nil {
			return fmt.Errorf("failed to run 'test' target on module %s: %s", modulePath, err)
		}
	} else {
		logger.Warnf("The 'really' flag was not set to true: not releasing anything.")
	}

	logger.Infof("Done.")

	return nil
}

func checkModuleHasMakefile(modulePath string) (string, error) {
	makefilePath := filepath.Join(modulePath, makeFile)
	info, err := os.Stat(makefilePath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("module %s has no Makefile. cannot release", modulePath)
	}
	if err != nil {
		return "", fmt.Errorf("problem while checking for Makefile in %s: %s", modulePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("module %s does not contain a correct Makefile", modulePath)
	}
	return makefilePath, nil
}

func runMakeTarget(modulePath string, target string, releaseTarget kaeter.ReleaseTarget) error {

	cmd := exec.Command("make", "-e", "VERSION=" + releaseTarget.Version, target)
	cmd.Dir = modulePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("call to make failed with error: %s", err)
	}
	return nil
}

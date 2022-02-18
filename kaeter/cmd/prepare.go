package cmd

import (
	"fmt"
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path"
	"path/filepath"
	"time"

	"github.com/open-ch/go-libs/fsutils"
	"github.com/open-ch/go-libs/gitshell"
	"github.com/spf13/cobra"
)

func init() {
	// For a SemVer versioned module, should the minor or major be bumped?
	var minor bool
	var major bool

	// Version passed via CLI
	var userProvidedVersion string

	// Branch, Tag or Commit to do a release from:
	var releaseFrom string

	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the release of the specified module.",
		Long: `Prepare the release of the specified module:

Based on the module's versions.yaml file and the flags passed to it, this command will:'
 - determine the next version to be released, using either SemVer of CalVer;
 - update the versions.yaml file for the relevant project
 - serialize the release plan to a commit`,
		Run: func(cmd *cobra.Command, args []string) {
			// The CLI is begging for a little refactor...
			// Fallback to gitMainBranch if no branch to release from was specified
			var baseBranch string
			if releaseFrom == "" {
				baseBranch = gitMainBranch
			} else {
				baseBranch = releaseFrom
			}
			err := runPrepare(major, minor, userProvidedVersion, baseBranch)
			if err != nil {
				logger.Errorf("Prepare failed: %s", err)
				os.Exit(1)
			}
		},
	}

	prepareCmd.Flags().BoolVar(&minor, "minor", false,
		`If set, and if the module is using SemVer, causes a bump in the minor version of the released module.
By default the build number is incremented.`)

	prepareCmd.Flags().BoolVar(&major, "major", false,
		`If set, and if the module is using SemVer, causes a bump in the major version of the released module.
By default the build number is incremented.`)

	prepareCmd.Flags().StringVar(&userProvidedVersion, "version", "",
		"If specified, this version will be used for the prepared release, instead of deriving one.")

	prepareCmd.Flags().StringVar(&releaseFrom, "releaseFrom", "",
		`If specified, use this identifier to resolve the commit id from which to do the release.
Can be a branch, a tag or a commit id.
Note that it is wise to release a commit that already exists in a remote.
Defaults to the value of the global --git-main-branch option.`)

	rootCmd.AddCommand(prepareCmd)
}

func runPrepare(bumpMajor bool, bumpMinor bool, userProvidedVersion string, releaseFrom string) error {
	releaseTargets := make([]kaeter.ReleaseTarget, len(modulePaths))

	refTime := time.Now()
	hash := gitshell.GitResolveRevision(repoRoot, releaseFrom)

	logger.Infof("Release based on %s, with commit id %s", releaseFrom, hash)

	for i, modulePath := range modulePaths {
		logger.Infof("Preparing release of module at %s", modulePath)
		absVersionsPath, err := pointToVersionsFile(modulePath)
		absModuleDir := filepath.Dir(absVersionsPath)
		if err != nil {
			return err
		}

		versions, err := kaeter.ReadFromFile(absVersionsPath)
		if err != nil {
			return err
		}
		logger.Infof("Module has identifier: %s", versions.ID)
		newReleaseMeta, err := versions.AddRelease(&refTime, bumpMajor, bumpMinor, userProvidedVersion, hash)
		if err != nil {
			return err
		}

		logger.Infof("Will prepare a release with version: %s", newReleaseMeta.Number.String())
		logger.Infof("Writing versions.yaml file at: %s", absVersionsPath)
		versions.SaveToFile(absVersionsPath)

		releaseTargets[i] = kaeter.ReleaseTarget{ModuleID: versions.ID, Version: newReleaseMeta.Number.String()}

		logger.Infof("Adding file to commit: %s", absVersionsPath)
		// Add the versions file we found, as it may be .yaml or .yml
		gitshell.GitAdd(absModuleDir, filepath.Base(absVersionsPath))

		logger.Infof("Done with release preparations for %s:%s", versions.ID, newReleaseMeta.Number.String())
	}

	releasePlan := &kaeter.ReleasePlan{Releases: releaseTargets}
	commitMsg, err := releasePlan.ToCommitMessage()
	if err != nil {
		return err
	}

	logger.Debugf("Writing Release Plan to commit with message:\n%s", commitMsg)

	logger.Infof("Committing staged changes...")
	gitshell.GitCommit(repoRoot, commitMsg)

	logger.Infof("Run 'git log' to check the commit message.")

	return nil
}

// pointToVersionsFile checks if the passed path is a directory, then:
//  - checks if there is a versions.yml or .yaml file, and appends the existing one to the abspath if so
//  - appends 'versions.yaml' to it if there is none.
func pointToVersionsFile(modulePath string) (string, error) {
	absModulePath, err := filepath.Abs(modulePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absModulePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		versionsFilesFound, err := fsutils.SearchByFileNameRegex(absModulePath, kaeter.VersionsFileNameRegex)
		if err != nil {
			return "", err
		}
		// Single match? Here we go
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
			// If there are multiple versions files in subdirs: fail
			return "", fmt.Errorf("found multiple versions file in: %s", modulePath)
		}
		// If no file exists yet we use the .yaml convention
		return filepath.Join(absModulePath, "versions.yaml"), nil
	}
	return absModulePath, nil
}

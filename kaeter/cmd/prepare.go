package cmd

import (
	"fmt"
	"os"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path/filepath"
	"time"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/open-ch/go-libs/fsutils"
	"github.com/spf13/cobra"
)

func init() {
	// For a SemVer versioned module, should the minor or major be bumped?
	var minor bool
	var major bool

	// Should the release plan be serialized to a commit.
	var commit bool

	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the release of the specified module.",
		Long: `Prepare the release of the specified module:

Based on the module's versions.yaml file and the flags passed to it, this command will:'
 - determine the next version to be released, using either SemVer of CalVer;
 - update the versions.yaml file for the relevant project
 - serialize the release plan to a commit`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runPrepare(major, minor)
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

	prepareCmd.Flags().BoolVar(&commit, "commit", true,
		`If set, saves the release plan to a commit message. The current git index  
is commited 'as-is': anything that was 'git add'ed before (without being commited) will be included,
but nothing else is added.`)

	rootCmd.AddCommand(prepareCmd)
}

func runPrepare(bumpMajor bool, bumpMinor bool) error {
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

	refTime := time.Now()

	hash := gitshell.GitResolveRevision(absModuleDir, gitMainBranch)

	logger.Infof("Release based on %s, with commit id %s", gitMainBranch, hash)
	newReleaseMeta, err := versions.AddRelease(&refTime, bumpMajor, bumpMinor, hash)
	if err != nil {
		return err
	}

	logger.Infof("Will prepare a release with version: %s", newReleaseMeta.Number.GetVersionString())
	logger.Infof("Writing versions.yaml file at: %s", absVersionsPath)
	versions.SaveToFile(absVersionsPath)

	rp := kaeter.SingleReleasePlan(versions.ID, newReleaseMeta.Number.GetVersionString())
	commitMsg, err := rp.ToCommitMessage()
	if err != nil {
		return err
	}

	logger.Debugf("Writing Release Plan to commit with message:\n%s", commitMsg)
	logger.Infof("Adding file to commit: %s", absVersionsPath)
	// Add the versions file we found, as it may be .yaml or .yml
	gitshell.GitAdd(absModuleDir, filepath.Base(absVersionsPath))

	logger.Infof("Committing staged changes...")
	gitshell.GitCommit(absModuleDir, commitMsg)

	logger.Infof("Done with release preparations for %s:%s", versions.ID, newReleaseMeta.Number.GetVersionString())
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
		// Find the versions file:
		matches, err := fsutils.SearchByFileNameRegex(absModulePath, versionsFileNameRegex)
		if err != nil {
			return "", err
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("found multiple versions file in: %s", modulePath)
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		// If no file exists yet we use the .yaml convention
		return filepath.Join(absModulePath, "versions.yaml"), nil
	}
	return absModulePath, nil
}

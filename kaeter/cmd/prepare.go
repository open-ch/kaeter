package cmd

import (
	"os"
	"github.com/open-ch/go-libs/fsutils"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
)

const refBranch = "origin/master"

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

Based on the module's versions.yml file and the flags passed to it, this command will:'
 - determine the next version to be released, using either SemVer of CalVer;
 - update the versions.yml file for the relevant project
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

	repo, wt, err := openRepoAndWorktree(absModuleDir)
	if err != nil {
		return err
	}
	// TODO make branch from which to read the commit id configurable
	hash, err := repo.ResolveRevision(refBranch)
	if err != nil {
		return err
	}

	logger.Infof("Release based on %s, with commit id %s", refBranch, hash.String())

	newReleaseMeta, err := versions.AddRelease(&refTime, bumpMajor, bumpMinor, hash.String())
	if err != nil {
		return err
	}
	logger.Infof("Will prepare a release with version: %s", newReleaseMeta.Number.GetVersionString())
	logger.Infof("Writing versions.yml file at: %s", absVersionsPath)
	versions.SaveToFile(absVersionsPath)

	rp := kaeter.SingleReleasePlan(versions.ID, newReleaseMeta.Number.GetVersionString())
	commitMsg, err := rp.ToCommitMessage()
	if err != nil {
		return err
	}
	logger.Debugf("Writing Release Plan to commit with message:\n%s", commitMsg)

	pathInRepo, err := normalizePathToRepoRoot(absModuleDir)

	if err != nil {
		return err
	}
	relativeVersionsFile := filepath.Join(pathInRepo, versionsFile)
	logger.Infof("Adding file to commit: %s", relativeVersionsFile)
	if _, err = wt.Add(filepath.Join(pathInRepo, versionsFile)); err != nil {
		return err
	}

	logger.Infof("Committing staged changes...")
	opts, err := buildCommitOptions()
	if err != nil {
		return err
	}
	if _, err = wt.Commit(commitMsg, opts); err != nil {
		return err
	}

	logger.Infof("Done with release preparations for %s:%s", versions.ID, newReleaseMeta.Number.GetVersionString())
	logger.Infof("Run 'git log' to check the commit message.")
	return nil
}

func buildCommitOptions() (*git.CommitOptions, error) {
	uname, err := gitconfig.Username()
	if err != nil {
		return nil, err
	}
	email, err := gitconfig.Email()
	if err != nil {
		return nil, err
	}
	return &git.CommitOptions{
		All: false,
		Author: &object.Signature{
			Name:  uname,
			Email: email,
			When:  time.Now(),
		},
	}, nil
}

// normalizePathToRepoRoot normalize the path to the repository's root.
// returns a relative path.
// Note: if used on a submodule, will return path relative to submodule.
func normalizePathToRepoRoot(path string) (string, error) {
	root, err := fsutils.SearchClosestParentContaining(path, ".git")
	if err != nil {
		return "", err
	}
	// It's important to remove the slash from the path: it confuses the hell out of go-git
	return strings.TrimPrefix(path, root+"/"), nil
}

// pointToVersionsFile checks if the passed path is a directory, and appends 'versions.yml' to it if so.
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
		return filepath.Join(absModulePath, versionsFile), nil
	}
	return absModulePath, nil
}

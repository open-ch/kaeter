package change

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	bazelshell "github.com/open-ch/go-libs/bazelshell"
)

// BazelChange contains the
// * Bazel source files in BazelSources
// * A flag whether the workspace files was changed in Workspace
// * Changed source files in SourceFiles
// * Changed packages in Packages
// * Change targets in Targets
type BazelChange struct {
	BazelSources []string
	Workspace bool
	SourceFiles []string
	BuildFiles []string
	Targets []string
	Packages []string
}

// BazelCheck performs change detection and returns all Bazel related change in a summary in BazelChange
func (d *Detector) BazelCheck(changes *Information) (c BazelChange) {

	// Determine the affected Bazel sources
	c.BazelSources = make([]string,0)
	d.detectBazelFiles(changes.Files.Modified, &c)
	d.detectBazelFiles(changes.Files.Added, &c)
	sort.Strings(c.BazelSources)

	// Check whether WORKSPACE was changed
	if contains(changes.Files.Modified, "WORKSPACE") {
		c.Workspace = true
	}

	// Get build files
	c.BuildFiles = make([]string,0)
	d.detectBuildfiles(changes.Files.Modified, &c)
	d.detectBuildfiles(changes.Files.Added, &c)
	sort.Strings(c.BuildFiles)

	// Get the affected source files
	c.SourceFiles = d.getSourceFiles(d.RootPath, changes.Files)
	sort.Strings(c.SourceFiles)

	// Get the affected targets
	c.Targets = d.getTargetsFromSourceFiles(d.RootPath, c.SourceFiles)
	sort.Strings(c.Targets)

	// Get the affected packages
	c.Packages = d.getPackagesFromTargets(c.Targets)

	return
}

// detectBazelFiles searches for Bazel source files in the repo
func (d *Detector) detectBazelFiles(list []string, bc *BazelChange) {
	for _, path := range list {
		fn := filepath.Ext(path)
		if  fn == ".bzl" {
			bc.BazelSources = append(bc.BazelSources, path)
		}
	}
}

// detectBuildfiles searches for Bazel build files in the repo
func (d *Detector) detectBuildfiles(list []string, bc *BazelChange) {
	for _, path := range list {
		fn := filepath.Base(path)
		if  fn == "BUILD" || fn == "BUILD.bazel" {
			bc.BuildFiles = append(bc.BuildFiles, path)
		}
	}
}

// getSourceFiles gets the list of affected sources files, files being a dependency of a Bazel rule.
func (d *Detector) getSourceFiles(gitRoot string, changes Files) []string {

	// Get the list of all Bazel source files in the repo
	query := "kind(\"source file\", deps(//...) except deps(//3rdparty/...))"
	d.Logger.Debugf("DetectorBazel: Query for source files: %s\n", query)
	bSourceFiles, err := bazelshell.Query(gitRoot, query, []string{"--keep_going", "--notool_deps"})
	if err != nil {
		d.Logger.Errorf("Failed to execute Bazel query for source files: %v", err)
		os.Exit(1)
	}

	// Post process and remove unwanted deps
	for i := 0; i < len(bSourceFiles); i++ {
		if strings.HasPrefix(bSourceFiles[i], "@") {
			// remove external sources, such as JARs
			bSourceFiles = remove(bSourceFiles, i)
			i--
		} else {
			// transform the source file path from the target style to a path style.
			bSourceFiles[i] = strings.Replace(strings.TrimLeft(bSourceFiles[i], "//"),":","/",1)
			d.Logger.Debugf("DetectorBazel: Bazel source file: %s", bSourceFiles[i])
		}
	}

	// Extract the intersection between the Git changes and the Bazel source files
	affectedSources := make([]string, 0)
	for _, gChangePath := range append(changes.Added, changes.Modified...) {
		if contains(bSourceFiles, gChangePath) {
			affectedSources = append(affectedSources, gChangePath)
			d.Logger.Debugf("DetectorBazel: Affected Bazel source file: %s", gChangePath)
		}
	}

	return affectedSources
}

// getTargetsFromSourceFiles derives the Bazel packages that depended on a set of source files
func (d *Detector) getTargetsFromSourceFiles(gitRoot string, sourceFiles []string) []string {
	// Prepare the Bazel query
	// Put the source file names in quotes
	sf := make([]string, len(sourceFiles))
	for i, p := range sourceFiles {
		sf[i] = fmt.Sprintf("'%s'", p)
	}
	// Query for the packages that depend on said source files
	query := fmt.Sprintf("rdeps(//..., set(%s)) except kind('source file', rdeps(//..., set(%s)))", strings.Join(sf, " "), strings.Join(sf, " "))
	d.Logger.Debugf("DetectorBazel: Query for packages: %s\n", query)
	targets, err := bazelshell.Query(gitRoot, query, []string{"--keep_going"})
	if err != nil {
		d.Logger.Errorf("Failed to execute Bazel query for targets: %v", err)
		os.Exit(1)
	}
	// Remove non-targets
	for i := 0; i < len(targets); i++ {
		if !strings.HasPrefix(targets[i], "//") {
			targets = remove(targets, i)
			i--
		} else {
			d.Logger.Debugf("DetectorBazel: Detected target: %s\n", targets[i])
		}
	}
	return targets
}

// getPackagesFromTargets extracts the affected packages
func (d *Detector) getPackagesFromTargets(targets []string) []string {
	packages := make([]string, 0)
	for _, t := range targets {
		s := strings.Split(t, ":")
		if ! contains(packages, s[0]) {
			packages = append(packages, s[0])
		}
	}
	return packages
}

package change

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// LabelCharacters is the list of valid character for a Bazel package
var LabelCharacters = "a-zA-Z0-9-_"

// RepoLabelRegex the regex to find labels of the current repository
var RepoLabelRegex = regexp.MustCompile(fmt.Sprintf("//([%s]+/)*[%s]+(:[%s]+){0,1}", LabelCharacters, LabelCharacters, LabelCharacters))

// PackageLabelRegex the regex to find labels of the current package
var PackageLabelRegex = regexp.MustCompile(fmt.Sprintf(":[%s]+", LabelCharacters))

// KaeterChange contains a map of changed Modules by ids
type KaeterChange struct {
	Modules map[string]modules.KaeterModule
}

// KaeterCheck attempts to find all Kaeter modules and infers based on the
// change set which module were altered
func (d *Detector) KaeterCheck(changes *Information) (kc KaeterChange, err error) {
	kc.Modules = make(map[string]modules.KaeterModule)
	allTouchedFiles := append(append(changes.Files.Added, changes.Files.Modified...), changes.Files.Removed...)

	// For each, resolve Bazel or non-Bazel targets
	for _, m := range d.KaeterModules {
		log.Debugf("DetectorKaeter: Inspected Module: %s", m.ModuleID)
		err = d.checkModuleForChanges(&m, &kc, allTouchedFiles)
		if err != nil {
			return kc, fmt.Errorf("error detecting changes for %s: %w", m.ModuleID, err)
		}
	}
	return kc, nil
}

func (d *Detector) checkModuleForChanges(m *modules.KaeterModule, kc *KaeterChange, allTouchedFiles []string) error {
	if m.ModuleType != "Makefile" {
		log.Warnf("DetectorKaeter: skipping unsupported non Makefile type module %s", m.ModuleID)
		return nil
	}

	// we also assume that any change affecting this folder or its subfolders affects the module
	modulePath := strings.TrimPrefix(m.ModulePath, d.RootPath+"/") + "/"
	for _, file := range allTouchedFiles {
		if strings.HasPrefix(file, modulePath) {
			log.Debugf("DetectorKaeter: File '%s' might affect module", file)
			kc.Modules[m.ModuleID] = *m
			// No need to go through the rest of the files, return fast and move to next module
			return nil
		}
		for _, dependency := range m.Dependencies {
			log.Debugf("DetectorKaeter: Dependency %s for Module: %s", dependency, m.ModuleID)
			if strings.HasPrefix(file, dependency) {
				log.Debugf("DetectorKaeter: File '%s' might affect module", file)
				kc.Modules[m.ModuleID] = *m
			}
		}
	}

	// TODO are we doing anything with these targets that we extract?
	// Was the idea that we would try to detect changes with a query after this as well?
	// We would need to take the intersection with the bazel partial detection to detect
	// and snapshot on bazel changes.
	// For now: short circuit if log level is above debug (makes change detection faster)
	if log.IsDebug() {
		return nil
	}

	absoluteModulePath := filepath.Join(d.RootPath, m.ModulePath)
	localBazelPackage := "/" + strings.TrimPrefix(m.ModulePath, d.RootPath)

	snapshotCommands, err := d.listMakeCommands(absoluteModulePath, "snapshot")
	if err != nil {
		return fmt.Errorf("failed identifying commands for %s: %w", m.ModuleID, err)
	}
	releaseCommands, err := d.listMakeCommands(absoluteModulePath, "release")
	if err != nil {
		return fmt.Errorf("failed identifying commands for %s: %w", m.ModuleID, err)
	}
	commands := append(
		snapshotCommands,
		releaseCommands...,
	)
	bazelTargets := d.extractBazelTargetsFromStrings(localBazelPackage, commands)

	log.Debugf("DetectorKaeter: Detected following bazel targets: %v", bazelTargets)

	return nil
}

// listMakeCommands extracts the commands executed by a Make target
func (*Detector) listMakeCommands(folder, target string) ([]string, error) {
	makefileName := "Makefile.kaeter"
	_, err := os.Stat(filepath.Join(folder, makefileName))
	if err != nil {
		makefileName = "Makefile"
	}

	cmd := exec.Command("make", "--file", makefileName, "--dry-run", target)
	log.Debugf("DetectorKaeter: Make command: %s", cmd.Args)
	cmd.Dir = folder
	var cmdOut []byte
	if cmdOut, err = cmd.CombinedOutput(); err != nil {
		return []string{}, fmt.Errorf("failed ready %s commands: %s\n%w", target, string(cmdOut), err)
	}

	return strings.Split(string(cmdOut), "\n"), nil
}

// extractBazelTargets extracts what looks like bazel targets from a bunch of strings.
func (*Detector) extractBazelTargetsFromStrings(packageName string, lines []string) (targets []string) {
	for _, line := range lines {
		// check whether the line contains bazel build or bazel run
		if strings.Contains(line, "bazel") && (strings.Contains(line, "build") || strings.Contains(line, "run")) {
			// search for full labels (package+target) //my/package:target
			if res := RepoLabelRegex.FindAllString(line, 1); len(res) != 0 && !contains(targets, res[0]) {
				targets = append(targets, res[0])
				continue
			}
			// search for target like :target, we then add the package name
			if res := PackageLabelRegex.FindAllString(line, 1); len(res) != 0 && !contains(targets, packageName+res[0]) {
				targets = append(targets, packageName+res[0])
			}
		}
	}
	sort.Strings(targets)
	return targets
}

package change

import (
	"os"
	"os/exec"
	"github.com/open-ch/kaeter/kaeter-ci/pkg/modules"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	bazelshell "github.com/open-ch/go-libs/bazelshell"
)

// KaeterChange contains a map of changed Modules by ids
type KaeterChange struct {
	Modules map[string]modules.KaeterModule
}

// KaeterCheck attempts to find all Kaeter modules and infers based on the
// change set which module were altered
func (d *Detector) KaeterCheck(changes *Information) (kc KaeterChange) {
	kc.Modules = make(map[string]modules.KaeterModule)
	kaeterModules, err := modules.GetKaeterModules(d.RootPath)
	if err != nil {
		d.Logger.Errorln("DetectorKaeter: Error fetching module list")
		d.Logger.Error(err)
		os.Exit(1)
	}
	allTouchedFiles := append(append(changes.Files.Added, changes.Files.Modified...), changes.Files.Removed...)

	// For each, resolve Bazel or non-Bazel targets
	for _, m := range kaeterModules {
		d.Logger.Debugf("DetectorKaeter: Inspected Module: %s", m.ModuleID)

		d.checkMakefileTypeForChanges(&m, &kc, changes, allTouchedFiles)
	}
	return
}

func (d *Detector) checkMakefileTypeForChanges(m *modules.KaeterModule, kc *KaeterChange, changes *Information, allTouchedFiles []string) {
	if m.ModuleType != "Makefile" {
		return
	}

	d.Logger.Debugf("DetectorKaeter: Module type is Makefile %s", m.ModuleID)

	absoluteModulePath := filepath.Join(d.RootPath, m.ModulePath)
	localBazelPackage := "/" + strings.TrimPrefix(m.ModulePath, d.RootPath)

	commands := append(
		d.listMakeCommands(absoluteModulePath, "snapshot"),
		d.listMakeCommands(absoluteModulePath, "release")...,
	)
	bazelTargets := d.extractBazelTargetsFromStrings(localBazelPackage, commands)
	d.Logger.Debugf("DetectorKaeter: Detected following bazel targets: %v", bazelTargets)

	// if Bazel targets, perform an rquery to check whether there are changes
	for _, t := range bazelTargets {
		if contains(changes.Bazel.Targets, t) {
			d.Logger.Debugf("DetectorKaeter: Target '%s' is affected by the change", t)
			kc.Modules[m.ModuleID] = *m
		}
	}
	// we also assume that any change affecting this folder or its subfolders
	// affects the module
	modulePath := strings.TrimPrefix(m.ModulePath, d.RootPath+"/") + "/"
	for _, file := range allTouchedFiles {
		if strings.HasPrefix(file, modulePath) {
			d.Logger.Debugf("DetectorKaeter: File '%s' might affect module", file)
			kc.Modules[m.ModuleID] = *m
		}
	}
}

// listMakeCommands extracts the commands executed by a Make target
func (d *Detector) listMakeCommands(folder, target string) []string {
	makefileName := "Makefile.kaeter"
	_, err := os.Stat(filepath.Join(folder, makefileName))
	if err != nil {
		makefileName = "Makefile"
	}

	cmd := exec.Command("make", "--file", makefileName, "--dry-run", target)
	d.Logger.Debugf("DetectorKaeter: Make command: %s", cmd.Args)
	cmd.Dir = folder
	var (
		cmdOut []byte
	)
	if cmdOut, err = cmd.Output(); err != nil {
		d.Logger.Errorf("Error reading the change files from Make %s in %s", makefileName, folder)
		d.Logger.Error(err)
		d.Logger.Error(os.Stderr)
		os.Exit(1)
	}

	return strings.Split(string(cmdOut), "\n")
}

// extractBazelTargets extracts what looks like bazel targets from a bunch of strings.
func (d *Detector) extractBazelTargetsFromStrings(packageName string, lines []string) (targets []string) {
	retr := regexp.MustCompile(bazelshell.RepoLabelRegex)
	reltr := regexp.MustCompile(bazelshell.PackageLabelRegex)
	for _, line := range lines {
		// check whether the line contains bazel build or bazel run
		if strings.Contains(line, "bazel") && (strings.Contains(line, "build") || strings.Contains(line, "run")) {
			// search for full labels (package+target) //my/package:target
			if res := retr.FindAllString(line, 1); len(res) != 0 && !contains(targets, res[0]) {
				targets = append(targets, res[0])
				continue
			}
			// search for target like :target, we then add the package name
			if res := reltr.FindAllString(line, 1); len(res) != 0 && !contains(targets, packageName+res[0]) {
				targets = append(targets, packageName+res[0])
			}
		}
	}
	sort.Strings(targets)
	return
}

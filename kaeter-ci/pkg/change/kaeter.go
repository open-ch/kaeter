package change

import (
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"github.com/open-ch/kaeter/kaeter/pkg/kaeter"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	bazelshell "github.com/open-ch/go-libs/bazelshell"
)

// KaeterChange contains a map of changed Modules by ids
type KaeterChange struct {
	Modules map[string]kaeterModule
}

type kaeterModule struct {
	ModuleID    string            `json:"id"`
	ModulePath  string            `json:"path"`
	ModuleType  string            `json:"type"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// KaeterCheck attempts to find all Kaeter modules and infers based on the
// change set which module were altered
func (d *Detector) KaeterCheck(changes *Information) (kc KaeterChange) {
	kc.Modules = make(map[string]kaeterModule)
	modules, err := d.getKaeterModules(d.RootPath)
	if err != nil {
		d.Logger.Errorln("DetectorKaeter: Error fetching module list")
		d.Logger.Error(err)
		os.Exit(1)
	}
	allTouchedFiles := append(append(changes.Files.Added, changes.Files.Modified...), changes.Files.Removed...)

	// For each, resolve Bazel or non-Bazel targets
	for _, m := range modules {
		d.Logger.Debugf("DetectorKaeter: Inspected Module: %s", m.ModuleID)

		d.checkMakefileTypeForChanges(&m, &kc, changes, allTouchedFiles)
	}
	return
}

func (d *Detector) checkMakefileTypeForChanges(m *kaeterModule, kc *KaeterChange, changes *Information, allTouchedFiles []string) {
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

// getKaeterModules searches the repo for all Kaeter modules. A Kaeter module is identified by having a
// versions.yaml file that is parseable by the Kaeter tooling.
func (d *Detector) getKaeterModules(gitRoot string) (modules []kaeterModule, err error) {
	// Extract the list of potential Kaeter modules by looking for all versions files.
	modulePath := make([]string, 0)
	err = filepath.WalkDir(gitRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		basename := filepath.Base(path)
		if basename == "versions.yaml" || basename == "versions.yml" {
			modulePath = append(modulePath, path)
		}
		return nil
	})

	if err != nil {
		return
	}

	// Try to parse the versions file, the parseable ones are Kaeter modules.
	for _, path := range modulePath {
		module, err := d.readKaeterModuleInfo(path)
		if err == nil {
			// error is logged by readKaeterModuleInfo, we skip over modules that do not load silently.
			modules = append(modules, module)
		}
	}
	return
}

func (d *Detector) readKaeterModuleInfo(versionsPath string) (module kaeterModule, err error) {
	data, err := ioutil.ReadFile(versionsPath)
	if err != nil {
		d.Logger.Errorf("DetectorKaeter: Could not read %s: %v", versionsPath, err)
		return
	}
	versions, err := kaeter.UnmarshalVersions(data)
	if err != nil {
		d.Logger.Errorf("DetectorKaeter: Could not parse %s: %v", versionsPath, err)
		return
	}
	modulePath, err := filepath.Rel(d.RootPath, filepath.Dir(versionsPath))
	if err != nil {
		d.Logger.Errorf("DetectorKaeter: Could find relative path in root (%s): %v", d.RootPath, err)
		return
	}
	module = kaeterModule{
		ModuleID:   versions.ID,
		ModulePath: modulePath,
		ModuleType: versions.ModuleType,
	}

	if versions.Metadata != nil && len(versions.Metadata.Annotations) > 0 {
		d.Logger.Errorf("Annotation Debug: available metadata: %v\n", versions.Metadata)
		module.Annotations = versions.Metadata.Annotations
	}

	return
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

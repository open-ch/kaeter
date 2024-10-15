package makefiles

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DetectModuleMakefile finds out if this modules has a
// Makefile.kaeter (preferred) or a Makefile.
func DetectModuleMakefile(modulePath string) (string, error) {
	makefileName := "Makefile.kaeter"
	makefilePath := filepath.Join(modulePath, makefileName)
	info, err := os.Stat(makefilePath)
	if err != nil {
		makefileName = "Makefile"
		makefilePath = filepath.Join(modulePath, makefileName)
		info, err = os.Stat(makefilePath)
	}
	if os.IsNotExist(err) {
		return "", fmt.Errorf("module %s has no Makefile", modulePath)
	}
	if err != nil {
		return "", fmt.Errorf("problem while checking for Makefile in %s: %w", modulePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("module %s Makefile cannot be a directory", modulePath)
	}
	return makefileName, nil
}

// RunTarget executes the given target with VERSION=version injected into the env
func RunTarget(modulePath, makefile, makeTarget, version string) error {
	versionArg := fmt.Sprintf("VERSION=%s", version)
	// We only execute make, the makefile and targets are in kaeter code, the version is the only outside argument
	// and it comes from the versions.yaml file, since it is a key in a map it is also limited and validated by kaeter
	// to a large extent.
	cmd := exec.Command("make", "--file", makefile, "--environment-overrides", versionArg, makeTarget)
	cmd.Dir = modulePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run() // TODO move to CombinedOutput() to avoid direct stdout output (see DryRunTarget())
	if err != nil {
		return fmt.Errorf("failed '%s' target on module %s: %w", makeTarget, modulePath, err)
	}
	return nil
}

// DryRunTarget gets the output of a make dryrun on a given target
func DryRunTarget(modulePath, makefile string, makeTargets []string) (string, error) {
	// About the inputs of exec.Command
	// - We're always executing make (from PATH)
	// - We run it from the context of the git repo kaeter is targeting
	// - the arguments we tack on at the end are hardcoded in kaeter and only contain target names
	cmd := exec.Command("make", append([]string{"--dry-run", "--file", makefile}, makeTargets...)...) //nolint:gosec
	cmd.Dir = modulePath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

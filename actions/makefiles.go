package actions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func detectModuleMakefile(modulePath string) (string, error) {
	makefileName := "Makefile.kaeter"
	makefilePath := filepath.Join(modulePath, makefileName)
	info, err := os.Stat(makefilePath)
	if err != nil {
		makefileName = "Makefile"
		makefilePath = filepath.Join(modulePath, makefileName)
		info, err = os.Stat(makefilePath)
	}
	if os.IsNotExist(err) {
		return "", fmt.Errorf("module %s has no Makefile. cannot release", modulePath)
	}
	if err != nil {
		return "", fmt.Errorf("problem while checking for Makefile in %s: %w", modulePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("module %s Makefile cannot be a directory", modulePath)
	}
	return makefileName, nil
}

func runMakeTarget(modulePath, makefile, makeTarget string, releaseTarget ReleaseTarget) error {
	versionArg := fmt.Sprintf("VERSION=%s", releaseTarget.Version)
	// We only execute make, the makefile and targets are in kaeter code, the version is the only outside argument
	// and it comes from the versions.yaml file, since it is a key in a map it is also limited and validated by kaeter
	// to a large extent.
	cmd := exec.Command("make", "--file", makefile, "-e", versionArg, makeTarget)
	cmd.Dir = modulePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed '%s' target on module %s: %w", makeTarget, modulePath, err)
	}
	return nil
}

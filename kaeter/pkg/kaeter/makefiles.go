package kaeter

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
		return "", fmt.Errorf("problem while checking for Makefile in %s: %s", modulePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("module %s Makefile cannot be a directory", modulePath)
	}
	return makefileName, nil
}

func runMakeTarget(modulePath string, makefile string, makeTarget string, releaseTarget ReleaseTarget) error {
	// Minor: we could pass in Version directly instead of releaseTarget
	cmd := exec.Command("make", "--file", makefile, "-e", "VERSION="+releaseTarget.Version, makeTarget)
	cmd.Dir = modulePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed '%s' target on module %s: %s", makeTarget, modulePath, err)
	}
	return nil
}

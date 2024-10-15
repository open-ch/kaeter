package lint

import (
	"fmt"
	"path/filepath"

	"github.com/open-ch/kaeter/makefiles"
)

func checkForValidMakefile(absModulePath string) error {
	makefileName, err := makefiles.DetectModuleMakefile(absModulePath)
	if err != nil {
		return fmt.Errorf("unable to locatate Makefile.kaeter or Makefile: %w", err)
	}

	requiredKaeterTargets := []string{
		"build",
		"test",
		"release",
	}
	output, err := makefiles.DryRunTarget(absModulePath, makefileName, requiredKaeterTargets)
	if err != nil {
		return fmt.Errorf("unable to validate %s targets in %s: %w\n%s", requiredKaeterTargets, filepath.Join(makefileName, absModulePath), err, output)
	}

	return nil
}

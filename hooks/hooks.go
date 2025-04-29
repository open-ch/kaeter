package hooks

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/open-ch/kaeter/modules"
)

const annotationPrefix = "open.ch/kaeter-hook/"

// HasHook checks if the module has an annotation defining the named hook
func HasHook(hookName string, module *modules.Versions) bool {
	if module == nil || module.Metadata == nil {
		return false
	}
	annotationName := annotationPrefix + hookName
	_, hookExists := module.Metadata.Annotations[annotationName]
	return hookExists
}

// RunHook executes the hook of the given name and returns it's output when
// successful. A list of arguments can be passed in.
//
// The value of the hook must be an executable with a path relative to the repository root.
func RunHook(hookName string, module *modules.Versions, repositoryRoot string, arguments []string) (string, error) {
	if module == nil || module.Metadata == nil {
		return "", errors.New("kaeter module has no annotations available")
	}
	annotationName := annotationPrefix + hookName
	hookPath, hookExists := module.Metadata.Annotations[annotationName]

	if !hookExists {
		return "", errors.New("kaeter module has no annotations available")
	}

	// Reject path traversal in hooks
	if strings.Contains(hookPath, "..") {
		// Note we might use https://go.dev/blog/osroot for a more accurate setup and avoid false positives.
		// and limit the path to only in repo.
		return "", errors.New("path traversal not allowed in hooks, use relative local paths only")
	}
	hookCmd := exec.Command(hookPath, arguments...)
	hookCmd.Dir = repositoryRoot
	output, err := hookCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execution of %s hook failed with the following error:\n%s\n%w", hookName, output, err)
	}
	return strings.TrimSpace(string(output)), nil
}

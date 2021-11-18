package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	kaeterChange "github.com/open-ch/kaeter/kaeter-ci/pkg/change"
)

func executeKaeterCI(kaeterPath, repoRoot, baseCommit, newCommit string) (info *kaeterChange.Information, err error) {
	changesetFile := "/tmp/changeset.json"
	var (
		cmdOut []byte
	)

	// bazel run //tools/kaeter-ci:cli -- check \
	//  --log-level debug \
	//  --path "${REPO_ROOT}" \
	//  --previous-commit "${CURRENT_HEAD}" \
	//  --latest-commit "${CURRENT_COMMIT}" \
	//  --output changeset.json
	cmd := exec.Command(kaeterPath,
		"check", "--log-level", "debug",
		"--path", repoRoot,
		"--previous-commit", baseCommit,
		"--latest-commit", newCommit,
		"--output", "/tmp/changeset.json",
	)
	fmt.Printf("Running command: %v\n", cmd.Args)
	if cmdOut, err = cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, string(cmdOut))
		fmt.Fprintln(os.Stderr, "There was an error executing kaeter-ci", err)
		return nil, err
	}

	// Now we read the result and return if for testing
	file, err := ioutil.ReadFile(changesetFile)
	if err != nil {
		return nil, err
	}
	info = &kaeterChange.Information{}
	fmt.Println(string(file))
	err = json.Unmarshal([]byte(file), info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

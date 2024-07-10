package lint

import (
	"fmt"
	"os"
	"github.com/open-ch/kaeter/modules"
	"regexp"
	"strings"
)

const expectedSpecChangelogReleaseFormat = "Expected format: * Day-of-Week Month Day Year usershort - Version-Release"

func findSpecFile(absModulePath string) (string, error) {
	files, err := os.ReadDir(absModulePath)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".spec") {
			return file.Name(), nil
		}
	}

	return "", fmt.Errorf(
		"no spec file found at %s",
		absModulePath,
	)
}

func checkSpecChangelog(changesPath string, versions *modules.Versions) error {
	changesRaw, err := os.ReadFile(changesPath)
	if err != nil {
		return fmt.Errorf("unable to load %s (%s)", changesPath, err.Error())
	}

	re := regexp.MustCompile(`%changelog`)
	if !re.Match(changesRaw) {
		return fmt.Errorf("%%changelog section not found in %s file", changesPath)
	}

	for _, releasedVersion := range versions.ReleasedVersions {
		if releasedVersion.CommitID == modules.InitRef {
			continue
		}
		version := releasedVersion.Number.String()

		// The format of entries in the change log can be described as:
		// '* Day-of-Week Month Day Year Name Surname <email> - Version-Release'
		// We mainly want to validate the version is present here and that the format
		// looks alright without being too strict.
		// Checks:
		// - Starts with *
		// - Inclules multiple words (date, author (optionally email))
		// - Include Version-Release (with a - before)
		//
		// Note: version in the spec will be in the 1.2.3-4 format:
		// - major: 1
		// - minor: 2
		// - patch: 3
		// - release: 4 (i.e. when the upstream version is the same but custom osix script changed)
		// RPM specs do NOT use 1.2.3+osag4, however 1.2.3-4 might be converted to an osix package of
		// 1.2.3+osag4.
		//
		// - (?m) enable multiline mode
		// https://pkg.go.dev/regexp/syntax
		re, err = regexp.Compile(`(?m)^\* [ .,<>@\w-]+ - ` + regexp.QuoteMeta(version) + `$`)
		if err != nil {
			return fmt.Errorf("failed to lookup version %s (%s)", version, err.Error())
		}

		if !re.Match(changesRaw) {
			return fmt.Errorf(
				"release notes for %s not found in specfile %s\n%s\nRegex: %s",
				version,
				changesPath,
				expectedSpecChangelogReleaseFormat,
				re,
			)
		}
	}

	return nil
}

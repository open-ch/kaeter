package lint

import (
	"fmt"
	"os"
	"github.com/open-ch/kaeter/modules"
	"regexp"
)

const expectedCHANGESReleaseFormat = "Expected format: vX.Y(.Z-...) dd.mm.yyyy (usr,sho,rts)"

func validateCHANGESFile(changesPath string, versions *modules.Versions) error {
	changesRaw, err := os.ReadFile(changesPath)
	if err != nil {
		return fmt.Errorf("unable to load %s (%s)", changesPath, err.Error())
	}

	for _, releasedVersion := range versions.ReleasedVersions {
		if releasedVersion.CommitID == modules.InitRef {
			continue // Ignore Kaeter's default INIT releases ("0.0.0: 1970-01-01T00:00:00Z|INIT")
		}

		version := releasedVersion.Number.String()
		// Matching of the released version in the CHANGES:
		// We expect that a line with the release has 3 components:
		// - a version (vX.Y, vX.Y.Z or other format)
		// - a date (mm.dd.yyyy)
		// - optional list of usernames involved
		//
		// Note that this regex targets specifically versions release via kaeter, so older releases
		// will not be bound by this check. Is this stricter for new release, maybe. Is that good,
		// maybe.
		//
		// - (?m) enable multiline mode
		// - (?:) non-capturing group
		// https://pkg.go.dev/regexp/syntax
		re, err := regexp.Compile(`(?m)^` + regexp.QuoteMeta(version) + `\s+\d{2}\.\d{2}\.\d{4}(?:\s+[,\w]+)?$`)
		if err != nil {
			return fmt.Errorf("failed to lookup version %s (%s)", version, err.Error())
			// Optional improvement we could try to match only the version and print it or suggest fixes
		}

		if !re.Match(changesRaw) {
			return fmt.Errorf(
				"release notes for %s not found in %s file\n%s",
				version,
				changesPath,
				expectedCHANGESReleaseFormat,
			)
		}
	}

	return nil
}

package actions

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

const dryrunMakefileContent = ".PHONY: build test\nbuild:\n\t@echo building\ntest:\n\t@echo testing"

func TestRunModuleRelease(t *testing.T) {
	testFolder := mocks.CreateTmpFolder(t)
	mocks.CreateMockFile(t, testFolder, "versions.yaml", "")
	mocks.CreateMockFile(t, testFolder, "Makefile", dryrunMakefileContent)
	moduleRelease := &ModuleRelease{
		CheckoutRestoreHash: "eeeeeeee",
		DryRun:              true,
		SkipCheckout:        true,
		RepositoryTrunk:     "origin/master",
		ReleaseTarget: ReleaseTarget{
			ModuleID: "ch.open:unit-test",
			Version:  "1.0.0",
		},
		VersionsYAMLPath: filepath.Join(testFolder, "versions.yaml"),
		VersionsData: &modules.Versions{
			ID:             "ch.open:unit-test",
			ModuleType:     "Makefile",
			VersioningType: "SemVer",
			ReleasedVersions: []*modules.VersionMetadata{
				{
					Number:    modules.NewVersion(1, 0, 0),
					Timestamp: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CommitID:  "deadbeef",
				},
			},
		},
	}

	err := RunModuleRelease(moduleRelease)

	assert.NoError(t, err)
}

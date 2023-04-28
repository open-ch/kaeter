package actions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/mocks"
	"github.com/open-ch/kaeter/kaeter/modules"
)

const dryrunMakefileContent = ".PHONY: build test\nbuild:\n\t@echo building\ntest:\n\t@echo testing"
const dummyMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target"
const errorMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo This target fails with error; exit 1"

func TestRunModuleRelease(t *testing.T) {
	testFolder := mocks.CreateTmpFolder(t)
	defer os.RemoveAll(testFolder)
	t.Logf("Temp test folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)", testFolder)
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
				&modules.VersionMetadata{
					Number:    &modules.VersionNumber{1, 0, 0},
					Timestamp: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CommitID:  "deadbeef",
				},
			},
		},
		Logger: log.New(),
	}

	err := RunModuleRelease(moduleRelease)

	assert.NoError(t, err)
}

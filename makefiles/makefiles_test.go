package makefiles

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

const dummyMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target with $(VERSION)"
const errorMakefileContent = ".PHONY: snapshot\nsnapshot:\n\t@echo This target fails with error; exit 1"

func TestDetectModuleMakefile(t *testing.T) {
	var tests = []struct {
		name             string
		makefiles        []string
		expectedMakefile string
		hasError         bool
	}{
		{
			name:             "Makefile only",
			makefiles:        []string{"Makefile"},
			expectedMakefile: "Makefile",
			hasError:         false,
		},
		{
			name:             "both Makefiles",
			makefiles:        []string{"Makefile", "Makefile.kaeter"},
			expectedMakefile: "Makefile.kaeter",
			hasError:         false,
		},
		{
			name:             "Makefile.kaeter only",
			makefiles:        []string{"Makefile.kaeter"},
			expectedMakefile: "Makefile.kaeter",
			hasError:         false,
		},
		{
			name:             "no Makefiles",
			makefiles:        []string{},
			expectedMakefile: "",
			hasError:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateTmpFolder(t)
			for _, makefileMock := range tc.makefiles {
				mocks.CreateMockFile(t, testFolder, makefileMock, mocks.EmptyMakefileContent)
			}

			makefile, err := DetectModuleMakefile(testFolder)

			if tc.hasError {
				assert.Error(t, err)
				assert.Equal(t, "", makefile)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedMakefile, makefile, "Failed detect expected Makefile")
			}
		})
	}
}

func TestRunTarget(t *testing.T) {
	var tests = []struct {
		name            string
		makefileName    string
		makefileContent string
		hasError        bool
	}{
		{
			name:            "Works with regular makefiles",
			makefileName:    "Makefile",
			makefileContent: dummyMakefileContent,
			hasError:        false,
		},
		{
			name:            "Works with Makefile.kaeter",
			makefileName:    "Makefile.kaeter",
			makefileContent: dummyMakefileContent,
			hasError:        false,
		},
		{
			name:            "Fails when make returns error",
			makefileName:    "Makefile.kaeter",
			makefileContent: errorMakefileContent,
			hasError:        true,
		},
		{
			name:            "Fails with no Makefile",
			makefileName:    "Makefile",
			makefileContent: "",
			hasError:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateTmpFolder(t)
			if tc.makefileContent != "" {
				mocks.CreateMockFile(t, testFolder, tc.makefileName, tc.makefileContent)
			}
			err := RunTarget(testFolder, tc.makefileName, "snapshot", "SNAPSHOT")

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDryRunTarget(t *testing.T) {
	// Note this test is a bit special in that it covers both our code but also
	// how we expect make to beheave so it's partially a contract test / documentation of what we expect
	var tests = []struct {
		name            string
		makefileContent string
		hasError        bool
	}{
		{
			name:            "Works with regular targets",
			makefileContent: "test:",
			hasError:        false,
		},
		{
			name:            "Works with phony targets",
			makefileContent: ".PHONY: build test release",
			hasError:        false,
		},
		{
			name:            "Fails when make returns error",
			makefileContent: "asdf",
			hasError:        true,
		},
		{
			name:            "Fails with no Makefile",
			makefileContent: "",
			hasError:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateTmpFolder(t)
			if tc.makefileContent != "" {
				mocks.CreateMockFile(t, testFolder, "Makefile", tc.makefileContent)
			}

			_, err := DryRunTarget(testFolder, "Makefile", []string{"test"})

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

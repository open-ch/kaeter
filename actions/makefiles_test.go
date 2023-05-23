package actions

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

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
			defer os.RemoveAll(testFolder)
			for _, makefileMock := range tc.makefiles {
				mocks.CreateMockFile(t, testFolder, makefileMock, dummyMakefileContent)
			}

			makefile, err := detectModuleMakefile(testFolder)

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

func TestRunMakeTarget(t *testing.T) {
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
			defer os.RemoveAll(testFolder)
			if tc.makefileContent != "" {
				mocks.CreateMockFile(t, testFolder, tc.makefileName, tc.makefileContent)
			}
			var target ReleaseTarget

			err := runMakeTarget(testFolder, tc.makefileName, "snapshot", target)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

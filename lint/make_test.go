package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
)

func TestCheckForValidMakefile(t *testing.T) {
	tests := []struct {
		name   string
		module mocks.KaeterModuleConfig
		valid  bool
	}{
		{
			name: "Accepts valid Makefile.kaeter",
			module: mocks.KaeterModuleConfig{
				Makefile:          mocks.EmptyMakefileContent,
				MakefileDotKaeter: true,
				VersionsYAML:      mocks.EmptyVersionsYAML,
			},
			valid: true,
		},
		{
			name: "Accepts valid Makefile",
			module: mocks.KaeterModuleConfig{
				Makefile:     mocks.EmptyMakefileContent,
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: true,
		},
		{
			name: "Acceps phony targets (Nothing to be done for ...)",
			module: mocks.KaeterModuleConfig{
				Makefile:     `.PHONY: build test snapshot release`,
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: true,
		},
		{
			name: "Fails if no Makefile is found",
			module: mocks.KaeterModuleConfig{
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: false,
		},
		{
			name: "Fails if Makefile is not valid",
			module: mocks.KaeterModuleConfig{
				Makefile:     `definitely;not###a_Makefile-thatisvalid`,
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: false,
		},
		{
			name: "Fail for missing build target",
			module: mocks.KaeterModuleConfig{
				Makefile: `test:
release:`,
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: false,
		},
		{
			name: "Fail for missing test target",
			module: mocks.KaeterModuleConfig{
				Makefile: `build:
release:`,
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: false,
		},
		{
			name: "Fail for missing release target",
			module: mocks.KaeterModuleConfig{
				Makefile: `build:

test:`,
				VersionsYAML: mocks.EmptyVersionsYAML,
			},
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modulePath, _ := mocks.CreateKaeterRepo(t, &tc.module)

			err := checkForValidMakefile(modulePath)

			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

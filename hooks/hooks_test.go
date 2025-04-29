package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/modules"
)

func TestHasHook(t *testing.T) {
	var tests = []struct {
		name         string
		kaeterModule *modules.Versions
		hasHooks     bool
	}{
		{
			name: "Empty module has no hooks",
		},
		{
			name: "Module with matching hook",
			kaeterModule: &modules.Versions{
				Metadata: &modules.Metadata{
					Annotations: map[string]string{
						"open.ch/kaeter-hook/test-hook": "path/to/hook/relative/to/repository/root",
					},
				},
			},
			hasHooks: true,
		},
		{
			name: "Module with other hook",
			kaeterModule: &modules.Versions{
				Metadata: &modules.Metadata{
					Annotations: map[string]string{
						"open.ch/kaeter-hook/other-hook": "path/to/hook/relative/to/repository/root",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hasHooks := HasHook("test-hook", tc.kaeterModule)

			assert.Equal(t, tc.hasHooks, hasHooks)
		})
	}
}

func TestRunHook(t *testing.T) {
	var genModule = func(hookPath string) *modules.Versions {
		return &modules.Versions{
			Metadata: &modules.Metadata{
				Annotations: map[string]string{
					"open.ch/kaeter-hook/test-hook": hookPath,
				},
			},
		}
	}

	var tests = []struct {
		name         string
		kaeterModule *modules.Versions
		expectError  bool
		expectOutput string
	}{
		{
			name:        "Empty module has no metadata or hooks",
			expectError: true,
		},
		{
			name:         "Module with metadata but no hooks",
			kaeterModule: &modules.Versions{Metadata: &modules.Metadata{Annotations: map[string]string{}}},
			expectError:  true,
		},
		{
			name:         "Module with matching hook but script does not exist",
			kaeterModule: genModule("testdata/non-existent.sh"),
			expectError:  true,
		},
		{
			name:         "Module with path traversal in module is not allowed",
			kaeterModule: genModule("../hooks/testdata/echo-args-hook.sh"),
			expectError:  true,
		},
		{
			name:         "Module with script that fails with error",
			kaeterModule: genModule("testdata/error-hook.sh"),
			expectError:  true,
		},
		{
			name:         "Module with script that returns a version",
			kaeterModule: genModule("testdata/valid-hook.sh"),
			expectOutput: "valid-result",
		},
		{
			name:         "Hook that handles arguments as expected ",
			kaeterModule: genModule("testdata/echo-args-hook.sh"),
			expectOutput: "echo-args one two 3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repositoryRoot := "."
			additionalArguments := []string{"one", "two", "3"}

			result, err := RunHook("test-hook", tc.kaeterModule, repositoryRoot, additionalArguments)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectOutput, result)
			}
		})
	}
}
